package benchmark

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/benchmark"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox/commands"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Phases of a health check, besides create and kill.
const (
	phaseFilesystem benchmark.Phase = "filesystem"
	phasePause      benchmark.Phase = "pause"
	phaseResume     benchmark.Phase = "resume"
)

// intranetURLTemplate builds the URL of a region's internal US3 (ufile)
// endpoint, which its sandboxes should reach over the intranet.
const intranetURLTemplate = "https://internal.s3-%s.ufileos.com/"

// intranetRegionAliases lists the regions whose US3 endpoint is named after
// a region code of its own.
var intranetRegionAliases = map[string]string{
	"cn-sh": "cn-sh2",
}

const (
	// probeTimeoutSec bounds one curl probe of an external address.
	probeTimeoutSec = 10

	// publicURLTimeout bounds how long the public URL check waits for the HTTP
	// server in the sandbox to answer through the gateway.
	publicURLTimeout = 30 * time.Second

	// publicURLPollInterval spaces the requests made while the server starts.
	publicURLPollInterval = 500 * time.Millisecond
)

// defaultInternetURL is the public URL checked by default: Apple's
// connectivity probe, served by a CDN close to every region, small and
// answering 200 without a redirect.
const defaultInternetURL = "https://www.apple.com/library/test/success.html"

// healthChecks names the checks --skip and --only select from, in the order
// they run. "pause" covers both pausing and resuming, since a resume needs a
// paused sandbox.
var healthChecks = []string{"exec", "filesystem", "internet", "intranet", "public-url", "pause"}

// healthOperation checks that the sandbox service works end to end: every
// sandbox is created, put through a series of checks and killed.
type healthOperation struct {
	runOptions

	urls          []string
	intranetURL   string
	publicURLPort int

	skip []string
	only []string
}

func (o *healthOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "health",
		Short: "Check that sandboxes work: exec, files, internet, intranet, public URL, pause and resume",
		Args:  cobra.NoArgs,
	}

	o.addFlags(c, 1, "Seconds each sandbox lives for, bounding any sandbox a failed kill leaves behind")
	c.Flags().StringArrayVarP(&o.urls, "url", "", []string{defaultInternetURL},
		"Public URL the sandbox must reach; repeat to check several")
	c.Flags().StringVarP(&o.intranetURL, "intranet-url", "", "",
		"Intranet URL the sandbox must get an HTTP response from, whatever its status (default: the region's internal US3 endpoint)")
	c.Flags().IntVarP(&o.publicURLPort, "public-url-port", "", 8000,
		"Sandbox port the public URL check serves HTTP on")

	checks := strings.Join(healthChecks, ", ")
	c.Flags().StringSliceVarP(&o.skip, "skip", "", nil, "Checks to leave out ("+checks+")")
	c.Flags().StringSliceVarP(&o.only, "only", "", nil, "Checks to run, leaving out the rest ("+checks+")")

	return c
}

func (o *healthOperation) validate() error {
	if err := o.runOptions.validate(); err != nil {
		return err
	}
	seen := make(map[string]bool, len(o.urls))
	for _, raw := range o.urls {
		if !isHTTPURL(raw) {
			return fmt.Errorf("invalid --url %q: must be an http or https URL", raw)
		}
		// Every URL is a phase of its own, named after it.
		if seen[raw] {
			return fmt.Errorf("duplicate --url %q", raw)
		}
		seen[raw] = true
	}
	if o.intranetURL != "" && !isHTTPURL(o.intranetURL) {
		return fmt.Errorf("invalid --intranet-url %q: must be an http or https URL", o.intranetURL)
	}
	if o.publicURLPort < 1 || o.publicURLPort > 65535 {
		return fmt.Errorf("invalid --public-url-port %d: must be between 1 and 65535", o.publicURLPort)
	}
	if len(o.skip) > 0 && len(o.only) > 0 {
		return errors.New("--skip and --only cannot be used together")
	}
	for _, flag := range []struct {
		name   string
		checks []string
	}{{"--skip", o.skip}, {"--only", o.only}} {
		for _, check := range flag.checks {
			if !slices.Contains(healthChecks, check) {
				return fmt.Errorf("invalid %s %q: must be one of %s", flag.name, check, strings.Join(healthChecks, ", "))
			}
		}
	}
	return nil
}

// isHTTPURL reports whether raw is an absolute http or https URL.
func isHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// selected reports whether --skip and --only leave check in.
func (o *healthOperation) selected(check string) bool {
	if len(o.only) > 0 {
		return slices.Contains(o.only, check)
	}
	return !slices.Contains(o.skip, check)
}

func (o *healthOperation) Run(ctx cmd.OperationContext) error {
	if err := o.validate(); err != nil {
		return err
	}

	sdk, err := newClient(ctx.Config)
	if err != nil {
		return err
	}

	intranetURL, err := o.resolveIntranetURL(ctx.Config)
	if err != nil {
		return err
	}

	return o.run(ctx, sdk, runSpec{
		name:  "health",
		steps: o.steps(sdk.Transport(), intranetURL),
	})
}

// resolveIntranetURL returns the URL the intranet check requests, or "" when
// the check is not selected. Without --intranet-url it is the US3 endpoint of
// the region the API calls go to, which has to be known.
func (o *healthOperation) resolveIntranetURL(cfg *config.Config) (string, error) {
	if !o.selected("intranet") || o.intranetURL != "" {
		return o.intranetURL, nil
	}
	region, ok := config.ResolveRegion(cfg)
	if !ok {
		return "", fmt.Errorf("cannot tell the region from domain %q to check its intranet; set the region or pass --intranet-url", cfg.Domain)
	}
	return intranetURLFor(region), nil
}

// intranetURLFor returns the URL of region's internal US3 endpoint.
func intranetURLFor(region string) string {
	if alias, ok := intranetRegionAliases[region]; ok {
		region = alias
	}
	return fmt.Sprintf(intranetURLTemplate, region)
}

// steps returns the steps of the selected checks, in the order of
// healthChecks.
func (o *healthOperation) steps(t *transport.Client, intranetURL string) []benchmark.Step[*benchSandbox] {
	var internet []benchmark.Step[*benchSandbox]
	for _, raw := range o.urls {
		internet = append(internet, probeStep(benchmark.Phase("internet "+raw), raw))
	}

	port := o.publicURLPort
	stepsOf := map[string][]benchmark.Step[*benchSandbox]{
		"exec":       {execStep("echo ok", "")},
		"filesystem": {{Phase: phaseFilesystem, Run: checkFilesystem}},
		"internet":   internet,
		"intranet":   {intranetStep(benchmark.Phase("intranet "+intranetURL), intranetURL)},
		"public-url": {{Phase: benchmark.Phase(fmt.Sprintf("public-url %d", port)), Run: func(ctx context.Context, sbx *benchSandbox) error {
			return checkPublicURL(ctx, t, sbx, port)
		}}},
		"pause": {
			{Phase: phasePause, Run: checkPause},
			{Phase: phaseResume, Run: checkResume},
		},
	}

	var steps []benchmark.Step[*benchSandbox]
	for _, check := range healthChecks {
		if o.selected(check) {
			steps = append(steps, stepsOf[check]...)
		}
	}
	return steps
}

// checkFilesystem writes a random token to a file through envd, reads it
// back and compares.
//
// The path is fixed: the random token in the content is what catches a
// mismatch, while a random path would put a different name in the error of
// every iteration, so the same failure would never group into one report row.
func checkFilesystem(ctx context.Context, sbx *benchSandbox) error {
	token := randomToken()
	const path = "/tmp/ucloud-sandbox-health"

	fs := sbx.Files()
	if _, err := fs.Write(ctx, path, token); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	got, err := fs.Read(ctx, path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if got != token {
		return fmt.Errorf("read %s: got %q, wrote %q", path, got, token)
	}

	// The sandbox is thrown away anyway; removing the file only keeps a long
	// lived one tidy.
	_ = fs.Remove(ctx, path)

	return nil
}

// probeStep checks that the sandbox reaches target over HTTP and gets 200
// back.
func probeStep(phase benchmark.Phase, target string) benchmark.Step[*benchSandbox] {
	return benchmark.Step[*benchSandbox]{
		Phase: phase,
		Run: func(ctx context.Context, sbx *benchSandbox) error {
			result, err := sbx.run(ctx, probeCommand(target), "")
			if err != nil {
				return fmt.Errorf("%s: %w", target, err)
			}
			if err := checkHTTPCode(result.Stdout); err != nil {
				return fmt.Errorf("%s: %w", target, err)
			}
			return nil
		},
	}
}

// probeCommand returns the curl invocation that prints the HTTP status code
// target answers with. curl fails, with a message on stderr, when the name
// does not resolve or the connection cannot be made.
func probeCommand(target string) string {
	return fmt.Sprintf("curl -sS -o /dev/null -m %d -w '%%{http_code}' %s", probeTimeoutSec, shellQuote(target))
}

// checkHTTPCode checks the status code a probe printed is 200. curl prints
// 000 when no response arrived. Redirects are not followed, so a 3xx fails.
func checkHTTPCode(stdout string) error {
	switch code := strings.TrimSpace(stdout); code {
	case "200":
		return nil
	case "", "000":
		return errors.New("no HTTP response")
	default:
		return fmt.Errorf("HTTP %s, want 200", code)
	}
}

// intranetStep checks that the sandbox gets an HTTP response from target.
// Its status does not matter: any response shows the sandbox reaches the
// service, and the default endpoint, US3, answers an anonymous request with
// 400. A failure tells how far the request got, so a slow name lookup can be
// told from a connection that never came.
func intranetStep(phase benchmark.Phase, target string) benchmark.Step[*benchSandbox] {
	return benchmark.Step[*benchSandbox]{
		Phase: phase,
		Run: func(ctx context.Context, sbx *benchSandbox) error {
			// A failed command still comes with what curl printed.
			result, err := sbx.run(ctx, intranetProbeCommand(target), "")
			var code, timeline string
			if result != nil {
				code, timeline = parseIntranetProbe(result.Stdout)
			}
			if timeline != "" {
				timeline = " (" + timeline + ")"
			}
			if err != nil {
				return fmt.Errorf("%s: %w%s", target, err, timeline)
			}
			if code == "" || code == "000" {
				return fmt.Errorf("%s: no HTTP response%s", target, timeline)
			}
			return nil
		},
	}
}

// intranetProbeCommand returns the curl invocation requesting target. It
// prints the status code, then the times, in seconds since the start, at
// which the name resolved, the connection was made and the first byte of the
// response arrived. curl prints them even when the request fails, with 000
// and 0 for what never happened.
func intranetProbeCommand(target string) string {
	return fmt.Sprintf("curl -sS -o /dev/null -m %d -w '%%{http_code} %%{time_namelookup} %%{time_connect} %%{time_starttransfer}' %s",
		probeTimeoutSec, shellQuote(target))
}

// parseIntranetProbe splits what intranetProbeCommand printed into the status
// code and a description of when each stage of the request was reached, "" if
// the output is not what it prints.
func parseIntranetProbe(stdout string) (code, timeline string) {
	fields := strings.Fields(stdout)
	if len(fields) != 4 {
		return "", ""
	}
	stages := []string{"resolved", "connected", "first byte"}
	parts := make([]string, 0, len(stages))
	for i, stage := range stages {
		sec, err := strconv.ParseFloat(fields[i+1], 64)
		switch {
		case err != nil:
			return fields[0], ""
		case sec == 0:
			parts = append(parts, stage+" -")
		default:
			parts = append(parts, fmt.Sprintf("%s %s", stage, time.Duration(sec*float64(time.Second)).Round(time.Millisecond)))
		}
	}
	return fields[0], strings.Join(parts, ", ")
}

// shellQuote quotes s as a single word for the shell envd runs commands in.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// checkPublicURL serves a random token over HTTP inside the sandbox and
// fetches it from outside through the sandbox's public URL for port, the one
// `sandbox host` reports, checking the gateway's inbound path.
func checkPublicURL(ctx context.Context, t *transport.Client, sbx *benchSandbox, port int) error {
	token := randomToken()
	// Fixed, like the filesystem check's path, so failures group.
	const dir = "/tmp/ucloud-sandbox-health-www"

	if _, err := sbx.Files().Write(ctx, dir+"/health", token); err != nil {
		return fmt.Errorf("write served file: %w", err)
	}

	server := fmt.Sprintf("python3 -m http.server %d --bind 0.0.0.0 --directory %s", port, shellQuote(dir))
	handle, err := sbx.Commands().Start(ctx, server, commands.Options{})
	if err != nil {
		return fmt.Errorf("start HTTP server: %w", err)
	}
	defer func() { _, _ = handle.Kill(context.WithoutCancel(ctx)) }()

	// A server that dies, python3 missing say, would otherwise only show up as
	// the gateway failing until the deadline.
	// The watcher ends with the check, even when killing the server fails and
	// its stream stays open.
	watchCtx, stopWatch := context.WithCancel(ctx)
	defer stopWatch()

	exited := make(chan string, 1)
	go func() {
		result, err := handle.Wait(watchCtx)
		switch {
		case err != nil:
			exited <- err.Error()
		case result != nil:
			exited <- fmt.Sprintf("exit code %d: %s", result.ExitCode, result.Stderr)
		default:
			exited <- "exited"
		}
	}()

	// SandboxHost resolves the host the way envd traffic does, falling back to
	// the configured domain when the reported one is not a domain.
	var reportedDomain string
	if sbx.Domain != nil {
		reportedDomain = *sbx.Domain
	}
	target := t.SandboxScheme() + "://" + t.SandboxHost(sbx.ID(), reportedDomain, port) + "/health"
	var trafficToken string
	if sbx.TrafficAccessToken != nil {
		trafficToken = *sbx.TrafficAccessToken
	}

	pollCtx, cancel := context.WithTimeout(ctx, publicURLTimeout)
	defer cancel()

	var last error
	for {
		body, err := fetch(pollCtx, t.HTTPClient(), target, trafficToken)
		switch {
		case err != nil:
			last = err
		case body != token:
			return fmt.Errorf("GET %s: got %q, served %q", target, body, token)
		default:
			return nil
		}

		select {
		case reason := <-exited:
			return fmt.Errorf("HTTP server in the sandbox stopped (%s); last attempt: %w", reason, last)
		case <-pollCtx.Done():
			return fmt.Errorf("GET %s: no answer within %s: %w", target, publicURLTimeout, last)
		case <-time.After(publicURLPollInterval):
		}
	}
}

// fetch GETs target and returns its body, failing on any status but 200.
func fetch(ctx context.Context, client *http.Client, target, trafficToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	if trafficToken != "" {
		req.Header.Set("E2B-Traffic-Access-Token", trafficToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: HTTP %d", target, resp.StatusCode)
	}
	return string(body), nil
}

// checkPause pauses the sandbox.
func checkPause(ctx context.Context, sbx *benchSandbox) error {
	if err := sbx.Pause(ctx, api.SandboxPauseRequest{}); err != nil {
		return err
	}
	sbx.paused = true
	return nil
}

// checkResume resumes the paused sandbox and runs a command in it, so the
// resumed sandbox is known to work and not only to be reported running.
func checkResume(ctx context.Context, sbx *benchSandbox) error {
	if !sbx.paused {
		return notAttempted("the sandbox did not pause")
	}

	resumed, err := sbx.svc.Connect(ctx, sbx.ID(), api.ConnectSandbox{Timeout: sbx.timeout})
	if err != nil {
		return err
	}
	sbx.Sandbox = resumed
	sbx.paused = false

	if _, err := sbx.run(ctx, "echo ok", ""); err != nil {
		return fmt.Errorf("exec after resume: %w", err)
	}
	return nil
}

// randomToken returns a random hex string, unique enough to name files and
// tell served content apart.
func randomToken() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
