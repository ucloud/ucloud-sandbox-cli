package benchmark

import (
	"context"
	"fmt"
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
}

// addFlags registers the shared flags on c. timeoutUsage describes what the
// sandbox timeout means for the command.
func (o *runOptions) addFlags(c *cobra.Command, timeoutUsage string) {
	c.Flags().StringVarP(&o.template, "template", "t", "system/base", "Template ID or alias to create sandboxes from")
	c.Flags().IntVarP(&o.count, "count", "n", 10, "Total number of iterations")
	c.Flags().IntVarP(&o.concurrency, "concurrency", "c", 1, "Number of iterations in flight at once")
	c.Flags().Int32VarP(&o.timeout, "timeout", "", 300, timeoutUsage)
	c.Flags().StringVarP(&o.format, "format", "f", list.FormatPretty, "Output format (pretty, json)")
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

// execOptions describe the command an iteration runs in its sandbox.
type execOptions struct {
	command string
	user    string
}

// run executes a benchmark named name and writes its report. phases selects
// which phases are timed; exec is only used when the exec phase is.
func (o *runOptions) run(ctx cmd.OperationContext, name string, exec execOptions, phases benchmark.LifecycleOptions) error {
	sdk, err := config.NewClient(ctx.Config, withoutRetries)
	if err != nil {
		return err
	}

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

	fmt.Fprintf(os.Stderr, "Running %s benchmark %s: template %s, %d iteration(s), concurrency %d\n",
		name, runID, o.template, o.count, o.concurrency)
	// Said up front, so a run that is killed outright still leaves a way to
	// find its sandboxes.
	fmt.Fprintf(os.Stderr, "Sandboxes are tagged %s=%s; clean them up with: %s\n", runIDMetadataKey, runID, cleanup)

	showProgress := term.IsTerminal(int(os.Stderr.Fd()))

	opts := phases
	opts.Count = o.count
	opts.Concurrency = o.concurrency
	opts.OnIteration = func(p benchmark.Progress) {
		if showProgress {
			fmt.Fprintf(os.Stderr, "\r[%d/%d] succeeded %d, failed %d", p.Done, p.Total, p.Succeeded, p.Failed)
		}
	}

	report := benchmark.RunLifecycle(runCtx, newCreator(sdk.Sandboxes(), runID, o.template, o.timeout, exec), opts)
	if showProgress && report.Started > 0 {
		fmt.Fprintln(os.Stderr)
	}

	view := newRunView(runID, o.template, o.concurrency, o.timeout, report)
	if phases.SkipKill && len(report.Phases) > 0 {
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
func newCreator(svc *sandbox.Service, runID, template string, timeout int32, exec execOptions) benchmark.CreateFunc {
	return func(ctx context.Context) (benchmark.Instance, error) {
		req := api.NewSandbox{
			TemplateID: template,
			Timeout:    &timeout,
			Metadata:   &api.SandboxMetadata{runIDMetadataKey: runID},
		}

		sbx, err := svc.Create(ctx, req, manageBy)
		if err != nil {
			return nil, err
		}

		return &sdkInstance{sbx: sbx, exec: exec}, nil
	}
}

// sdkInstance adapts an SDK sandbox to benchmark.Instance.
type sdkInstance struct {
	sbx  *sandbox.Sandbox
	exec execOptions
}

func (i *sdkInstance) ID() string {
	return i.sbx.SandboxID
}

func (i *sdkInstance) Exec(ctx context.Context) error {
	// Run reports a non-zero exit as an error, which counts as a failed exec.
	_, err := i.sbx.Commands().User(i.exec.user).Run(ctx, i.exec.command, commands.Options{})
	return err
}

// Kill relies on the SDK's contract for Sandbox.Kill: (false, nil) is returned
// only when DELETE /sandboxes/{id} answers 404, and every other failure comes
// back as an error. That is what lets !killed be reported as not found.
func (i *sdkInstance) Kill(ctx context.Context) error {
	killed, err := i.sbx.Kill(ctx)
	if err != nil {
		return err
	}
	if !killed {
		return benchmark.ErrSandboxNotFound
	}
	return nil
}
