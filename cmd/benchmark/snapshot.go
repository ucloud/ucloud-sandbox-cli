package benchmark

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/benchmark"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox/commands"
)

// Phases of a snapshot benchmark, besides create and kill.
const (
	phaseWrite          benchmark.Phase = "write"
	phaseSnapshot       benchmark.Phase = "snapshot"
	phaseRestore        benchmark.Phase = "restore"
	phaseRestoreVerify  benchmark.Phase = "restore-verify"
	phaseFork           benchmark.Phase = "fork"
	phaseForkVerify     benchmark.Phase = "fork-verify"
	phaseSnapshotDelete benchmark.Phase = "snapshot-delete"

	// phaseCleanup kills the copies. It is not timed, and has no row of its
	// own; it only shows up when it fails.
	phaseCleanup benchmark.Phase = "cleanup"
)

// copyPlaceholder stands in for a copy's sandbox ID in error messages, so the
// same failure in different copies groups together.
const copyPlaceholder = "<copy>"

// tokenFile is where the write step leaves its token. It is in the user's
// home, expanded by the sandbox's shell, so the snapshot is sure to keep it.
const tokenFile = `"$HOME/.ucloud-sandbox-bench-token"`

// snapshotState is what the snapshot steps of one iteration hand to the ones
// after them. An empty field means the step producing it did not succeed.
type snapshotState struct {
	token string

	// snapshotID boots the snapshot; snapshotName, the name it was taken
	// under, is what deletes it.
	snapshotID   string
	snapshotName string

	restored *sandbox.Sandbox
	forkedID string
}

// snapshotOperation times snapshotting a sandbox, booting a sandbox from the
// snapshot and forking it, checking that both copies keep its files.
type snapshotOperation struct {
	runOptions
}

func (o *snapshotOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "snapshot",
		Short: "Measure snapshot, restore, fork and snapshot delete latency, checking copies keep the sandbox's files",
		Args:  cobra.NoArgs,
	}

	o.addFlags(c, 3, "Seconds each sandbox lives for, bounding any sandbox a failed kill leaves behind")

	return c
}

func (o *snapshotOperation) Run(ctx cmd.OperationContext) error {
	if err := o.validate(); err != nil {
		return err
	}

	sdk, err := newClient(ctx.Config)
	if err != nil {
		return err
	}

	leaked := &idList{}

	return o.run(ctx, sdk, runSpec{
		name:            "snapshot",
		steps:           snapshotSteps(leaked),
		snapshots:       true,
		leakedSnapshots: leaked.list,
	})
}

// snapshotNamePrefix starts the name of every snapshot a run takes, so the
// ones it fails to delete can be told apart in `snapshot list`.
func snapshotNamePrefix(runID string) string {
	return "bench-" + runID + "-"
}

// snapshotSteps returns the steps of a snapshot iteration. The copies are
// killed before the snapshot is deleted, so the delete is timed as it is
// normally made, with nothing booted from the snapshot still running.
// Snapshots that cannot be deleted are added to leaked.
func snapshotSteps(leaked *idList) []benchmark.Step[*benchSandbox] {
	return []benchmark.Step[*benchSandbox]{
		{Phase: phaseWrite, Run: writeToken},
		{Phase: phaseSnapshot, Run: takeSnapshot},
		{Phase: phaseRestore, Run: restoreSnapshot},
		{Phase: phaseRestoreVerify, Run: verifyRestored},
		{Phase: phaseFork, Run: forkSandbox},
		{Phase: phaseForkVerify, Run: verifyForked},
		{Phase: phaseCleanup, Run: killCopies, Untimed: true},
		{Phase: phaseSnapshotDelete, Run: func(ctx context.Context, sbx *benchSandbox) error {
			return deleteSnapshot(ctx, sbx, leaked)
		}},
	}
}

// notAttempted fails a step that depends on one that did not succeed.
func notAttempted(reason string) error {
	return errors.New("not attempted: " + reason)
}

// writeToken leaves a random token in the sandbox for the copies to carry.
func writeToken(ctx context.Context, sbx *benchSandbox) error {
	token := randomToken()
	if _, err := sbx.run(ctx, fmt.Sprintf("printf %%s %s > %s", token, tokenFile), ""); err != nil {
		return err
	}
	sbx.snap.token = token
	return nil
}

// takeSnapshot snapshots the sandbox under a name tied to the run.
func takeSnapshot(ctx context.Context, sbx *benchSandbox) error {
	name := snapshotNamePrefix(sbx.runID) + sbx.ID()
	info, err := sbx.svc.CreateSnapshot(ctx, sbx.ID(), api.SandboxSnapshotRequest{Name: &name})
	if err != nil {
		return err
	}
	sbx.snap.snapshotID = info.SnapshotID
	sbx.snap.snapshotName = name
	return nil
}

// restoreSnapshot boots a sandbox from the snapshot.
func restoreSnapshot(ctx context.Context, sbx *benchSandbox) error {
	if sbx.snap.snapshotID == "" {
		return notAttempted("no snapshot was taken")
	}
	restored, err := sbx.svc.Create(ctx, sandboxRequest(sbx.snap.snapshotID, sbx.runID, sbx.timeout), manageBy)
	if err != nil {
		return err
	}
	sbx.snap.restored = restored
	return nil
}

// verifyRestored checks the sandbox booted from the snapshot holds the token.
func verifyRestored(ctx context.Context, sbx *benchSandbox) error {
	if sbx.snap.restored == nil {
		return notAttempted("no sandbox was restored")
	}
	return withoutID(verifyToken(ctx, sbx.snap.restored, sbx.snap.token), sbx.snap.restored.SandboxID)
}

// forkSandbox forks one copy of the sandbox.
func forkSandbox(ctx context.Context, sbx *benchSandbox) error {
	count := int32(1)
	results, err := sbx.svc.Fork(ctx, sbx.ID(), api.SandboxForkRequest{Count: &count, Timeout: &sbx.timeout})
	if err != nil {
		return err
	}
	if len(results) != 1 {
		return fmt.Errorf("asked for 1 fork, got %d results", len(results))
	}
	switch result := results[0]; {
	case result.Error != nil:
		return fmt.Errorf("fork failed: %s", result.Error.Message)
	case result.Sandbox == nil:
		return errors.New("fork reported neither a sandbox nor an error")
	default:
		sbx.snap.forkedID = result.Sandbox.SandboxID
		return nil
	}
}

// verifyForked connects to the fork and checks it holds the token.
func verifyForked(ctx context.Context, sbx *benchSandbox) error {
	if sbx.snap.forkedID == "" {
		return notAttempted("no sandbox was forked")
	}
	forked, err := sbx.svc.Connect(ctx, sbx.snap.forkedID, api.ConnectSandbox{Timeout: sbx.timeout})
	if err != nil {
		return withoutID(fmt.Errorf("connect to fork: %w", err), sbx.snap.forkedID)
	}
	return withoutID(verifyToken(ctx, forked, sbx.snap.token), sbx.snap.forkedID)
}

// verifyToken checks that copy holds the token written to the original.
func verifyToken(ctx context.Context, copy *sandbox.Sandbox, token string) error {
	if token == "" {
		return notAttempted("no token was written")
	}
	result, err := copy.Commands().Run(ctx, "cat "+tokenFile, commands.Options{})
	if err != nil {
		return err
	}
	// The tokens are random, so they stay out of the message to let the same
	// failure in different iterations group together.
	if strings.TrimSpace(result.Stdout) != token {
		return errors.New("token mismatch: the copy does not hold what was written to the original")
	}
	return nil
}

// withoutID replaces a copy's sandbox ID in err's message with a placeholder.
// The runner only knows the original sandbox's ID, so it cannot do this for
// the copies.
func withoutID(err error, id string) error {
	if err == nil || id == "" || !strings.Contains(err.Error(), id) {
		return err
	}
	return errors.New(strings.ReplaceAll(err.Error(), id, copyPlaceholder))
}

// deleteSnapshot deletes the snapshot, adding its name to leaked when that
// fails.
//
// The snapshot is deleted by the name it was taken under. DELETE
// /templates/{id} does not find a named snapshot by the ID it was created
// with ("<team>/<name>:<tag>"), nor without the tag, and answers 404.
//
// A 404 is a failure too, and leaks the snapshot: unlike a sandbox, a
// snapshot never expires, so one taken moments ago that cannot be found is
// one the request missed, not one that is gone.
func deleteSnapshot(ctx context.Context, sbx *benchSandbox, leaked *idList) error {
	name := sbx.snap.snapshotName
	if name == "" {
		return notAttempted("no snapshot was taken")
	}
	deleted, err := sbx.svc.DeleteSnapshot(ctx, name)
	if err != nil {
		leaked.add(name)
		return err
	}
	// Like Sandbox.Kill, DeleteSnapshot reports a 404 as (false, nil).
	if !deleted {
		leaked.add(name)
		return errors.New("snapshot not found (HTTP 404)")
	}
	return nil
}

// killCopies kills the restored and forked sandboxes, reporting the ones it
// could not kill in a benchmark.LeakedError. A copy that is already gone needs
// no cleanup.
func killCopies(ctx context.Context, sbx *benchSandbox) error {
	var ids []string
	if sbx.snap.restored != nil {
		ids = append(ids, sbx.snap.restored.SandboxID)
	}
	if sbx.snap.forkedID != "" {
		ids = append(ids, sbx.snap.forkedID)
	}

	var leaked []string
	var first error
	for _, id := range ids {
		if _, err := sbx.svc.Kill(ctx, id); err != nil {
			leaked = append(leaked, id)
			if first == nil {
				first = withoutID(err, id)
			}
		}
	}
	if len(leaked) > 0 {
		return &benchmark.LeakedError{IDs: leaked, Err: first}
	}
	return nil
}

// idList collects IDs from concurrent iterations.
type idList struct {
	mu  sync.Mutex
	ids []string
}

func (l *idList) add(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ids = append(l.ids, id)
}

func (l *idList) list() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.ids...)
}
