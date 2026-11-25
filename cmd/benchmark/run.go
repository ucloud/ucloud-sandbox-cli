package benchmark

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/benchmark"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox/commands"
	"golang.org/x/term"
)

// runIDMetadataKey tags every sandbox a run creates, so the ones it leaves
// behind can be found and cleaned up.
const runIDMetadataKey = "benchmark"

// manageBy is the SDK manage-by marker set on every sandbox a benchmark
// creates, telling them apart from sandboxes opened by other products.
const manageBy = "sandbox-bench"

// runOptions are the flags shared by every benchmark command.
type runOptions struct {
	template    string
	count       int
	concurrency int
	timeout     int32
	format      string
	breakdown   bool
}

// addFlags registers the shared flags on c. defaultCount is the number of
// iterations run when --count is not given, and timeoutUsage describes what
// the sandbox timeout means for the command.
func (o *runOptions) addFlags(c *cobra.Command, defaultCount int, timeoutUsage string) {
	c.Flags().StringVarP(&o.template, "template", "t", "system/base", "Template ID or alias to create sandboxes from")
	c.Flags().IntVarP(&o.count, "count", "n", defaultCount, "Total number of iterations")
	c.Flags().IntVarP(&o.concurrency, "concurrency", "c", 1, "Number of iterations in flight at once")
	c.Flags().Int32VarP(&o.timeout, "timeout", "", 300, timeoutUsage)
	c.Flags().StringVarP(&o.format, "format", "f", list.FormatPretty, "Output format (pretty, json)")
	c.Flags().BoolVarP(&o.breakdown, "breakdown", "", false,
		"Show each phase's full latency, split into connection setup (DNS, dial, TLS) and service time")
}

func (o *runOptions) validate() error {
	if o.count < 1 {
		return fmt.Errorf("invalid --count %d: must be 1 or greater", o.count)
	}
	if o.concurrency < 1 {
		return fmt.Errorf("invalid --concurrency %d: must be 1 or greater", o.concurrency)
	}
	if o.timeout < 1 {
		return fmt.Errorf("invalid --timeout %d: must be 1 or greater", o.timeout)
	}
	switch o.format {
	case list.FormatPretty, list.FormatJSON:
	default:
		return fmt.Errorf("invalid --format %q: must be %s or %s", o.format, list.FormatPretty, list.FormatJSON)
	}
	return nil
}

// runSpec describes what a benchmark command times on every sandbox.
type runSpec struct {
	// name labels the run in its progress output.
	name string

	// steps run on every sandbox between create and kill.
	steps []benchmark.Step[*benchSandbox]

	// skipKill leaves the sandboxes running until their timeout.
	skipKill bool

	// snapshots marks a run that takes snapshots, which outlive it unless
	// deleted. leakedSnapshots, when set, returns the ones it failed to delete.
	snapshots       bool
	leakedSnapshots func() []string
}

// newClient returns the SDK client benchmarks run on: the configured one,
// with the retry of rate-limited requests turned off.
func newClient(cfg *config.Config) (*client.Client, error) {
	return config.NewClient(cfg, withoutRetries)
}

// run executes a benchmark and writes its report.
func (o *runOptions) run(ctx cmd.OperationContext, sdk *client.Client, spec runSpec) error {
	runID := newRunID(time.Now())
	cleanup := cleanupCommand(runID)

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The first Ctrl+C stops new iterations and waits for the ones in flight
	// to finish; restoring the default handling lets a second one force the
	// process out. finished is closed before the deferred stop runs, so a run
	// that ends normally is not reported as interrupted.
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-runCtx.Done():
			stop()
			fmt.Fprintf(os.Stderr, "\nInterrupted: waiting for in-flight iterations to finish (Ctrl+C again to force quit, then clean up with: %s)...\n", cleanup)
		case <-finished:
		}
	}()

	o.writeHeader(os.Stderr, spec, runID)

	showProgress := term.IsTerminal(int(os.Stderr.Fd()))

	opts := benchmark.Options[*benchSandbox]{
		Count:       o.count,
		Concurrency: o.concurrency,
		Steps:       spec.steps,
		SkipKill:    spec.skipKill,
		OnIteration: func(p benchmark.Progress) {
			if showProgress {
				fmt.Fprintf(os.Stderr, "\r[%d/%d] succeeded %d, failed %d", p.Done, p.Total, p.Succeeded, p.Failed)
			}
		},
	}

	report := benchmark.Run(runCtx, newCreator(sdk.Sandboxes(), runID, o.template, o.timeout), opts)
	if showProgress && report.Started > 0 {
		fmt.Fprintln(os.Stderr)
	}

	view := newRunView(runID, o.template, o.concurrency, o.timeout, report)
	if spec.leakedSnapshots != nil {
		view.LeakedSnapshots = append(view.LeakedSnapshots, spec.leakedSnapshots()...)
	}
	view.breakdown = o.breakdown
	if spec.skipKill && len(report.Phases) > 0 {
		view.LeftRunning = report.Phases[0].Stats.Count
	}

	if o.format == list.FormatJSON {
		if err := cmd.ShowJSON(view); err != nil {
			return err
		}
	} else if err := view.writePretty(os.Stdout); err != nil {
		return err
	}

	return exitError(report)
}

// writeHeader describes the run about to start. The cleanup command is given
// up front, so a run that is killed outright still leaves a way to find its
// sandboxes.
func (o *runOptions) writeHeader(w io.Writer, spec runSpec, runID string) {
	fmt.Fprintf(w, "Running %s benchmark %s\n", spec.name, runID)
	fmt.Fprintf(w, "  Template:     %s\n", o.template)
	fmt.Fprintf(w, "  Iterations:   %d (concurrency %d)\n", o.count, o.concurrency)
	fmt.Fprintf(w, "  Sandbox tag:  %s=%s\n", runIDMetadataKey, runID)
	if spec.snapshots {
		fmt.Fprintf(w, "  Snapshots:    named %s<sandbox-id>\n", snapshotNamePrefix(runID))
	}
	fmt.Fprintf(w, "  Cleanup:      %s\n", cleanupCommand(runID))
	if spec.snapshots {
		fmt.Fprintf(w, "                ucloud-sandbox-cli snapshot delete <snapshot-name...>\n")
	}
	fmt.Fprintln(w)
}

// exitError turns a report into the command's error, so scripts can tell a
// clean run from one that failed or was cut short.
func exitError(r benchmark.Report) error {
	switch {
	case r.Interrupted:
		return fmt.Errorf("interrupted: %d of %d iteration(s) started, %d failed", r.Started, r.Requested, r.Failed)
	case r.Failed > 0:
		return fmt.Errorf("%d of %d iteration(s) failed", r.Failed, r.Started)
	default:
		return nil
	}
}

// newRunID returns the ID tagging the sandboxes of one run. The start time
// keeps it readable; the random suffix keeps runs started within the same
// second apart, so cleaning up one run never touches another.
func newRunID(now time.Time) string {
	return fmt.Sprintf("%s-%06x", now.Format("20060102-150405"), rand.Uint32()&0xffffff)
}

// cleanupCommand returns the command that kills every sandbox of a run.
func cleanupCommand(runID string) string {
	return fmt.Sprintf("ucloud-sandbox-cli sandbox kill --all -m %s=%s", runIDMetadataKey, runID)
}

// withoutRetries turns off the SDK's retry of rate-limited (429) requests. A
// retry waits out the Retry-After inside the call being timed, so it would
// pass rate limiting off as latency; failing instead shows it in the errors.
func withoutRetries(opts *client.Options) {
	retries := 0
	opts.Retries = &retries
}

// newCreator returns the CreateFunc that starts the sandboxes of a run.
func newCreator(svc *sandbox.Service, runID, template string, timeout int32) benchmark.CreateFunc[*benchSandbox] {
	return func(ctx context.Context) (*benchSandbox, error) {
		created, err := svc.Create(ctx, sandboxRequest(template, runID, timeout), manageBy)
		if err != nil {
			return nil, err
		}

		return &benchSandbox{Sandbox: created, svc: svc, runID: runID, timeout: timeout}, nil
	}
}

// sandboxRequest returns the request creating a sandbox of a run from
// template, which may also be a snapshot ID.
func sandboxRequest(template, runID string, timeout int32) api.NewSandbox {
	return api.NewSandbox{
		TemplateID: template,
		Timeout:    &timeout,
		Metadata:   &api.SandboxMetadata{runIDMetadataKey: runID},
	}
}

// benchSandbox is the sandbox one iteration works on. It embeds the SDK
// sandbox, so the steps use its files, commands and fields directly.
type benchSandbox struct {
	*sandbox.Sandbox

	// svc, runID and timeout let the steps start, connect to and tag further
	// sandboxes of the run.
	svc     *sandbox.Service
	runID   string
	timeout int32

	// paused records that the pause step succeeded, so the resume step can
	// tell a resume from a connect to a sandbox that never paused.
	paused bool

	// snap is what the snapshot steps hand to the ones after them.
	snap snapshotState
}

func (sbx *benchSandbox) ID() string {
	return sbx.SandboxID
}

// run runs command in the sandbox as user. A non-zero exit is an error.
func (sbx *benchSandbox) run(ctx context.Context, command, user string) (*commands.Result, error) {
	return sbx.Commands().User(user).Run(ctx, command, commands.Options{})
}

// Kill shadows the SDK's Sandbox.Kill to fit benchmark.Sandbox. It relies on
// the SDK's contract: (false, nil) is returned only when DELETE
// /sandboxes/{id} answers 404, and every other failure comes back as an
// error. That is what lets !killed be reported as not found.
func (sbx *benchSandbox) Kill(ctx context.Context) error {
	killed, err := sbx.Sandbox.Kill(ctx)
	if err != nil {
		return err
	}
	if !killed {
		return benchmark.ErrSandboxNotFound
	}
	return nil
}

// execStep runs command in the sandbox; a non-zero exit fails it.
func execStep(command, user string) benchmark.Step[*benchSandbox] {
	return benchmark.Step[*benchSandbox]{
		Phase: benchmark.PhaseExec,
		Run: func(ctx context.Context, sbx *benchSandbox) error {
			_, err := sbx.run(ctx, command, user)
			return err
		},
	}
}
