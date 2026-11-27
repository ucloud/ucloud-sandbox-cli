package benchmark

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/internal/benchmark"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
)

// fakeControlPlane answers the control-plane requests of the snapshot steps
// and records what it was asked.
type fakeControlPlane struct {
	mu sync.Mutex

	creates   []api.NewSandbox
	snapshots []api.SandboxSnapshotRequest
	killed    []string
	deleted   []string

	// Statuses to answer with instead of success.
	forkError      string
	deleteSnapshot int
	killStatus     int
}

func (f *fakeControlPlane) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	sandboxJSON := func(id string) string {
		return `{"sandboxID":"` + id + `","templateID":"tpl","clientID":"c","envdVersion":"0.2.0"}`
	}
	created := func(body string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(body))
	}

	path := r.URL.Path
	switch {
	case r.Method == http.MethodPost && path == "/sandboxes":
		var req api.NewSandbox
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.creates = append(f.creates, req)
		created(sandboxJSON("sbx-" + string(rune('a'+len(f.creates)-1))))

	case r.Method == http.MethodPost && strings.HasSuffix(path, "/snapshots"):
		var req api.SandboxSnapshotRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.snapshots = append(f.snapshots, req)
		created(`{"snapshotID":"team/` + *req.Name + `:default","names":["team/` + *req.Name + `:default"]}`)

	case r.Method == http.MethodPost && strings.HasSuffix(path, "/fork"):
		if f.forkError != "" {
			created(`[{"error":{"code":500,"message":"` + f.forkError + `"}}]`)
			return
		}
		created(`[{"sandbox":` + sandboxJSON("sbx-fork") + `}]`)

	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/templates/"):
		ref := strings.TrimPrefix(path, "/templates/")
		f.deleted = append(f.deleted, ref)
		// As the service does, a named snapshot is only found by its bare
		// name, not by the "<team>/<name>:<tag>" ID it was created with.
		if strings.ContainsAny(ref, "/:") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(statusOr(f.deleteSnapshot, http.StatusNoContent))

	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/sandboxes/"):
		f.killed = append(f.killed, strings.TrimPrefix(path, "/sandboxes/"))
		w.WriteHeader(statusOr(f.killStatus, http.StatusNoContent))

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func statusOr(status, fallback int) int {
	if status != 0 {
		return status
	}
	return fallback
}

// newFakeSandbox creates a benchSandbox of run-1 against fake.
func newFakeSandbox(t *testing.T, fake *fakeControlPlane) *benchSandbox {
	t.Helper()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	sdk, err := client.New(client.Options{APIKey: "test-key", APIURL: srv.URL})
	require.NoError(t, err)

	sbx, err := newCreator(sdk.Sandboxes(), "run-1", "system/base", 300)(context.Background())
	require.NoError(t, err)
	return sbx
}

func TestSnapshotStepsAgainstFakeService(t *testing.T) {
	fake := &fakeControlPlane{}
	sbx := newFakeSandbox(t, fake)
	ctx := context.Background()
	leaked := &idList{}

	require.NoError(t, takeSnapshot(ctx, sbx))
	require.Len(t, fake.snapshots, 1)
	assert.Equal(t, "bench-run-1-sbx-a", *fake.snapshots[0].Name, "named after the run and the sandbox")
	assert.Equal(t, "team/bench-run-1-sbx-a:default", sbx.snap.snapshotID)

	require.NoError(t, restoreSnapshot(ctx, sbx))
	require.Len(t, fake.creates, 2)
	restore := fake.creates[1]
	assert.Equal(t, "team/bench-run-1-sbx-a:default", restore.TemplateID, "booted from the snapshot")
	assert.Equal(t, "run-1", (*restore.Metadata)[runIDMetadataKey], "tagged with the run, so cleanup finds it")
	require.NotNil(t, sbx.snap.restored)

	require.NoError(t, forkSandbox(ctx, sbx))
	assert.Equal(t, "sbx-fork", sbx.snap.forkedID)

	require.NoError(t, deleteSnapshot(ctx, sbx, leaked))
	assert.Equal(t, []string{"bench-run-1-sbx-a"}, fake.deleted, "deleted by name, not by ID")
	assert.Empty(t, leaked.list())

	require.NoError(t, killCopies(ctx, sbx))
	assert.Equal(t, []string{"sbx-b", "sbx-fork"}, fake.killed, "both copies are killed, the original is not")
}

func TestForkSandboxReportsFailedFork(t *testing.T) {
	sbx := newFakeSandbox(t, &fakeControlPlane{forkError: "no capacity"})

	err := forkSandbox(context.Background(), sbx)
	require.Error(t, err)
	assert.Equal(t, "fork failed: no capacity", err.Error())
	assert.Empty(t, sbx.snap.forkedID)
}

func TestDeleteSnapshotFailures(t *testing.T) {
	taken := func(sbx *benchSandbox) {
		sbx.snap.snapshotID = "team/bench-run-1-sbx-a:default"
		sbx.snap.snapshotName = "bench-run-1-sbx-a"
	}

	t.Run("server error leaks the snapshot", func(t *testing.T) {
		sbx := newFakeSandbox(t, &fakeControlPlane{deleteSnapshot: http.StatusInternalServerError})
		taken(sbx)
		leaked := &idList{}

		require.Error(t, deleteSnapshot(context.Background(), sbx, leaked))
		assert.Equal(t, []string{"bench-run-1-sbx-a"}, leaked.list(), "leaked by the name that deletes it")
	})

	t.Run("not found leaks the snapshot too", func(t *testing.T) {
		sbx := newFakeSandbox(t, &fakeControlPlane{deleteSnapshot: http.StatusNotFound})
		taken(sbx)
		leaked := &idList{}

		err := deleteSnapshot(context.Background(), sbx, leaked)
		require.Error(t, err)
		assert.Equal(t, "snapshot not found (HTTP 404)", err.Error())
		assert.Equal(t, []string{"bench-run-1-sbx-a"}, leaked.list(),
			"a snapshot taken moments ago does not expire; a 404 means the request missed it")
	})
}

func TestKillCopiesReportsWhatItCannotKill(t *testing.T) {
	fake := &fakeControlPlane{}
	sbx := newFakeSandbox(t, fake)
	sbx.snap.forkedID = "sbx-fork"
	fake.killStatus = http.StatusInternalServerError

	err := killCopies(context.Background(), sbx)
	var leaked *benchmark.LeakedError
	require.ErrorAs(t, err, &leaked)
	assert.Equal(t, []string{"sbx-fork"}, leaked.IDs)
	assert.NotContains(t, err.Error(), "sbx-fork", "the message groups across copies")
}

func TestSnapshotStepsNotAttempted(t *testing.T) {
	ctx := context.Background()
	sbx := &benchSandbox{}

	for name, run := range map[string]func() error{
		"restore":         func() error { return restoreSnapshot(ctx, sbx) },
		"restore-verify":  func() error { return verifyRestored(ctx, sbx) },
		"fork-verify":     func() error { return verifyForked(ctx, sbx) },
		"snapshot-delete": func() error { return deleteSnapshot(ctx, sbx, &idList{}) },
	} {
		err := run()
		require.Error(t, err, name)
		assert.True(t, strings.HasPrefix(err.Error(), "not attempted: "), "%s: %v", name, err)
		assert.ErrorIs(t, err, errNotAttempted, name)
	}

	assert.NoError(t, killCopies(ctx, sbx), "nothing to clean up")
}

func TestSnapshotStepOrder(t *testing.T) {
	var phases []benchmark.Phase
	var untimed []benchmark.Phase
	for _, s := range snapshotSteps(&idList{}) {
		phases = append(phases, s.Phase)
		if s.Untimed {
			untimed = append(untimed, s.Phase)
		}
	}
	assert.Equal(t, []benchmark.Phase{
		"write", "snapshot", "restore", "restore-verify", "fork", "fork-verify", "cleanup", "snapshot-delete",
	}, phases, "the copies are gone before the snapshot is deleted")
	assert.Equal(t, []benchmark.Phase{"cleanup"}, untimed)
}

func TestWithoutID(t *testing.T) {
	assert.NoError(t, withoutID(nil, "sbx-1"))

	err := errors.New(`Post "https://49983-sbx-1.example.com/process.Process/Start": refused`)
	assert.Equal(t, `Post "https://49983-<copy>.example.com/process.Process/Start": refused`, withoutID(err, "sbx-1").Error())

	unrelated := errors.New("timeout")
	assert.Same(t, unrelated, withoutID(unrelated, "sbx-1"), "an error without the ID is kept as it is")
	assert.Same(t, unrelated, withoutID(unrelated, ""))
}

func TestSnapshotDefaults(t *testing.T) {
	op := &snapshotOperation{}
	require.NoError(t, op.Command().ParseFlags(nil))
	assert.Equal(t, 3, op.count)
	assert.NoError(t, op.validate())
}

func TestRunViewSnapshotCleanup(t *testing.T) {
	render := func(v runView) string {
		var b strings.Builder
		require.NoError(t, v.writePretty(&b))
		return b.String()
	}

	clean := newRunView("run-1", "system/base", 1, 300, benchmark.Report{
		Phases: []benchmark.PhaseReport{{Phase: phaseSnapshot}},
	})
	assert.NotContains(t, render(clean), "snapshot delete")

	data, err := json.Marshal(clean)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"leaked_snapshots":[]`)

	leaky := clean
	leaky.LeakedSnapshots = []string{"bench-run-1-a", "bench-run-1-b"}
	out := render(leaky)
	assert.Contains(t, out, "2 snapshot(s) could not be deleted")
	assert.Contains(t, out, "Delete them with: ucloud-sandbox-cli snapshot delete bench-run-1-a bench-run-1-b")

	failed := newRunView("run-1", "system/base", 1, 300, benchmark.Report{
		Phases: []benchmark.PhaseReport{{Phase: phaseSnapshot, Failed: 1}},
	})
	out = render(failed)
	assert.Contains(t, out, "1 snapshot(s) failed; the service may still have taken some of them.")
	assert.Contains(t, out, "snapshots whose names contain bench-run-1- in: ucloud-sandbox-cli snapshot list")

	failure := func(phase benchmark.Phase, err error) benchmark.IterationResult {
		return benchmark.IterationResult{Phases: []benchmark.PhaseResult{{Phase: phase, Err: err}}}
	}
	copies := newRunView("run-1", "system/base", 1, 300, benchmark.Report{
		Phases: []benchmark.PhaseReport{{Phase: phaseRestore, Failed: 2}, {Phase: phaseFork, Failed: 2}},
		Iterations: []benchmark.IterationResult{
			failure(phaseRestore, errors.New("HTTP 500")),
			failure(phaseRestore, notAttempted("no snapshot was taken")),
			failure(phaseFork, errors.New("timeout")),
			failure(phaseFork, errors.New("timeout")),
		},
	})
	out = render(copies)
	assert.Contains(t, out, "3 restore(s) or fork(s) failed; the service may still have started some of those sandboxes.",
		"a restore that was never attempted started nothing")
	assert.Contains(t, out, "Clean them up with: ucloud-sandbox-cli sandbox kill --all -m benchmark=run-1")

	notSent := newRunView("run-1", "system/base", 1, 300, benchmark.Report{
		Phases:     []benchmark.PhaseReport{{Phase: phaseSnapshot}, {Phase: phaseRestore, Failed: 1}},
		Iterations: []benchmark.IterationResult{failure(phaseRestore, notAttempted("no snapshot was taken"))},
	})
	assert.NotContains(t, render(notSent), "restore(s) or fork(s) failed")
}

func TestWriteHeaderSnapshots(t *testing.T) {
	op := &runOptions{template: "system/base", count: 3, concurrency: 1}

	var b strings.Builder
	op.writeHeader(&b, runSpec{name: "snapshot", snapshots: true}, "run-1")

	assert.Equal(t, `Running snapshot benchmark run-1
  Template:     system/base
  Iterations:   3 (concurrency 1)
  Sandbox tag:  benchmark=run-1
  Snapshots:    named bench-run-1-<sandbox-id>
  Cleanup:      ucloud-sandbox-cli sandbox kill --all -m benchmark=run-1
                ucloud-sandbox-cli snapshot delete <snapshot-name...>

`, b.String())
}
