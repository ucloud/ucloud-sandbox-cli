package benchmark

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/internal/benchmark"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
)

func parseLifecycle(t *testing.T, args ...string) *lifecycleOperation {
	t.Helper()
	op := &lifecycleOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))
	return op
}

func TestLifecycleDefaults(t *testing.T) {
	op := parseLifecycle(t)

	assert.Equal(t, "system/base", op.template)
	assert.Equal(t, 10, op.count)
	assert.Equal(t, 1, op.concurrency)
	assert.Equal(t, "echo ok", op.command)
	assert.EqualValues(t, 300, op.timeout)
	assert.Equal(t, "pretty", op.format)
	assert.NoError(t, op.validate())
}

func TestLifecycleValidate(t *testing.T) {
	tests := []struct {
		name string
		args []string
		err  string
	}{
		{"zero count", []string{"-n", "0"}, "--count"},
		{"zero concurrency", []string{"-c", "0"}, "--concurrency"},
		{"zero timeout", []string{"--timeout", "0"}, "--timeout"},
		{"empty command", []string{"--cmd", ""}, "--cmd"},
		{"bad format", []string{"-f", "yaml"}, "--format"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseLifecycle(t, tt.args...).validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}

	assert.NoError(t, parseLifecycle(t, "--cmd", "", "--skip-exec").validate(),
		"an empty command is fine when exec is skipped")
	assert.NoError(t, parseLifecycle(t, "-t", "my-tpl", "-n", "50", "-c", "8", "-f", "json").validate())
}

func sampleReport() benchmark.Report {
	stats := benchmark.Stats{
		Count: 2,
		Avg:   1500 * time.Millisecond,
		Min:   time.Second,
		P50:   time.Second,
		P95:   2 * time.Second,
		P99:   2 * time.Second,
		Max:   2 * time.Second,
	}
	return benchmark.Report{
		Requested: 3,
		Started:   3,
		Succeeded: 2,
		Failed:    1,
		Elapsed:   4 * time.Second,
		Phases: []benchmark.PhaseReport{
			{
				Phase:   benchmark.PhaseCreate,
				Stats:   stats,
				Connect: benchmark.Stats{Count: 2, Avg: 100 * time.Millisecond, Min: 0, P50: 0, P95: 200 * time.Millisecond, P99: 200 * time.Millisecond, Max: 200 * time.Millisecond},
				Service: benchmark.Stats{Count: 2, Avg: 1400 * time.Millisecond, Min: time.Second, P50: time.Second, P95: 1800 * time.Millisecond, P99: 1800 * time.Millisecond, Max: 1800 * time.Millisecond},
			},
			{Phase: benchmark.PhaseExec, Failed: 1},
		},
		Errors: []benchmark.ErrorCount{
			{Phase: benchmark.PhaseExec, Message: "exit status 1", Count: 1},
		},
		Leaked: []string{"sbx-1"},
	}
}

func TestLifecycleViewJSON(t *testing.T) {
	v := newRunView("run-1", "system/base", 2, 300, sampleReport())

	data, err := json.Marshal(v)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "run-1", decoded["run_id"])
	assert.InDelta(t, 4000, decoded["elapsed_ms"], 1e-9)
	assert.InDelta(t, 0.5, decoded["rate_per_sec"], 1e-9)

	phases := decoded["phases"].([]any)
	require.Len(t, phases, 2)
	create := phases[0].(map[string]any)
	assert.Equal(t, "create", create["phase"])
	assert.InDelta(t, 1500, create["avg_ms"], 1e-9)
	assert.InDelta(t, 2000, create["p99_ms"], 1e-9)
	assert.InDelta(t, 200, create["connect"].(map[string]any)["p99_ms"], 1e-9)
	assert.InDelta(t, 1400, create["service"].(map[string]any)["avg_ms"], 1e-9)

	assert.Equal(t, []any{"sbx-1"}, decoded["leaked"])
}

func TestLifecycleViewJSONEmptyListsAreArrays(t *testing.T) {
	v := newRunView("run-1", "system/base", 1, 300, benchmark.Report{})

	data, err := json.Marshal(v)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"errors":[]`)
	assert.Contains(t, string(data), `"leaked":[]`)
	assert.Contains(t, string(data), `"iterations":[]`)
}

func TestRunViewJSONIterations(t *testing.T) {
	r := sampleReport()
	r.Iterations = []benchmark.IterationResult{
		{
			Seq:       1,
			Start:     1500 * time.Microsecond,
			SandboxID: "sbx-1",
			Phases: []benchmark.PhaseResult{
				{Phase: benchmark.PhaseCreate, Duration: time.Second, Connect: 200 * time.Millisecond},
				{Phase: benchmark.PhaseExec, Err: errors.New("exit status 1:\n  sbx-1 said no")},
				{Phase: benchmark.PhaseKill, Duration: 50 * time.Millisecond},
			},
			Leaked: []string{"sbx-1"},
		},
		{
			Seq:    2,
			Start:  3 * time.Millisecond,
			Phases: []benchmark.PhaseResult{{Phase: benchmark.PhaseCreate, Err: errors.New("quota exceeded")}},
		},
	}

	data, err := json.Marshal(newRunView("run-1", "system/base", 2, 300, r))
	require.NoError(t, err)

	var decoded struct {
		Iterations []map[string]any `json:"iterations"`
	}
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Len(t, decoded.Iterations, 2)

	first := decoded.Iterations[0]
	assert.InDelta(t, 1, first["seq"], 0)
	assert.Equal(t, "sbx-1", first["sandbox_id"])
	assert.InDelta(t, 1.5, first["start_offset_ms"], 1e-9)
	assert.Equal(t, false, first["succeeded"])
	assert.Equal(t, []any{"sbx-1"}, first["leaked"])
	assert.Equal(t, []any{
		map[string]any{"phase": "create", "duration_ms": 1000.0, "connect_ms": 200.0},
		// The sandbox ID is kept, unlike in the grouped errors.
		map[string]any{"phase": "exec", "error": "exit status 1: sbx-1 said no"},
		// A zero connect time is still written.
		map[string]any{"phase": "kill", "duration_ms": 50.0, "connect_ms": 0.0},
	}, first["phases"])

	second := decoded.Iterations[1]
	assert.Equal(t, "", second["sandbox_id"], "a failed create has no sandbox")
	assert.Equal(t, []any{}, second["leaked"])
	assert.Equal(t, []any{map[string]any{"phase": "create", "error": "quota exceeded"}}, second["phases"])
}

func TestRunViewPrettyOmitsIterations(t *testing.T) {
	r := sampleReport()
	r.Iterations = []benchmark.IterationResult{{Seq: 1, SandboxID: "sbx-iteration"}}

	var b strings.Builder
	require.NoError(t, newRunView("run-1", "system/base", 2, 300, r).writePretty(&b))

	assert.NotContains(t, b.String(), "sbx-iteration")
}

func TestLifecycleViewPretty(t *testing.T) {
	v := newRunView("run-1", "system/base", 2, 300, sampleReport())
	v.breakdown = true

	var b strings.Builder
	require.NoError(t, v.writePretty(&b))
	out := b.String()

	assert.Contains(t, out, "Benchmark run-1 completed: template system/base, concurrency 2")
	assert.Contains(t, out, "Iterations: 3/3 started, 2 succeeded, 1 failed")
	assert.Contains(t, out, "Elapsed: 4s, rate: 0.50 sandbox/s")
	assert.Contains(t, out, "Latency per phase, split into connection setup (connect) and service time (service):\n+")
	assert.Contains(t, out, "| create    | 2         | 0      | 1.5s  | 1s  | 1s  | 2s    | 2s    | 2s    |")
	assert.Contains(t, out, "|   connect | -         | -      | 100ms | 0s  | 0s  | 200ms | 200ms | 200ms |")
	assert.Contains(t, out, "|   service | -         | -      | 1.4s  | 1s  | 1s  | 1.8s  | 1.8s  | 1.8s  |")
	assert.NotContains(t, out, "Page:")

	// A phase without a single success shows no latencies and no breakdown.
	assert.Contains(t, out, "| exec      | 0         | 1      | -     | -   | -   | -     | -     | -     |\n+")

	assert.Contains(t, out, "Errors:")
	assert.Contains(t, out, "exit status 1")
	assert.Contains(t, out, "sandbox kill --all -m benchmark=run-1")
}

func TestRunViewShowsServiceTimeByDefault(t *testing.T) {
	var b strings.Builder
	require.NoError(t, newRunView("run-1", "system/base", 2, 300, sampleReport()).writePretty(&b))
	out := b.String()

	// The create row carries the service time (1.4s avg), not the full
	// latency (1.5s), and nothing is broken out under it.
	assert.Contains(t, out, "| create | 2         | 0      | 1.4s | 1s  | 1s  | 1.8s | 1.8s | 1.8s |")
	assert.Contains(t, out, "| exec   | 0         | 1      | -    | -   | -   | -    | -    | -    |")
	assert.Contains(t, out, "Service time per phase, excluding connection setup (DNS, dial, TLS); --breakdown shows the full latency:\n+")
	assert.NotContains(t, out, "| connect")
	assert.NotContains(t, out, "| service")
}

func TestOutputFlag(t *testing.T) {
	op := parseLifecycle(t)
	assert.Empty(t, op.output, "no report file by default")

	op = parseLifecycle(t, "-o", "/tmp/report.json")
	assert.Equal(t, "/tmp/report.json", op.output)
	assert.NoError(t, op.validate(), "the path is checked when it is written")

	assert.Equal(t, "/tmp/report.json", parseLifecycle(t, "--output", "/tmp/report.json").output)
}

func TestReportInterruptionAnnounces(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var b strings.Builder
	stopped := false
	reportInterruption(ctx, make(chan struct{}), func() { stopped = true }, &b, "cleanup-cmd")

	assert.True(t, stopped, "the default signal handling is restored, so a second Ctrl+C quits")
	assert.Contains(t, b.String(), "Interrupted: waiting for in-flight iterations to finish")
	assert.Contains(t, b.String(), "cleanup-cmd")
}

func TestReportInterruptionSilentWhenRunFinished(t *testing.T) {
	// Both channels are ready, as when a run finishes in the instant its
	// context is cancelled: the run was not interrupted.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	finished := make(chan struct{})
	close(finished)

	var b strings.Builder
	stopped := false
	reportInterruption(ctx, finished, func() { stopped = true }, &b, "cleanup-cmd")

	assert.Empty(t, b.String())
	assert.False(t, stopped)
}

func TestReportInterruptionSilentWithoutCancellation(t *testing.T) {
	finished := make(chan struct{})
	close(finished)

	var b strings.Builder
	reportInterruption(context.Background(), finished, func() {}, &b, "cleanup-cmd")

	assert.Empty(t, b.String())
}

func TestCheckReportFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	require.NoError(t, checkReportFile(path))
	_, err := os.Stat(path)
	require.NoError(t, err, "the file is created up front")

	assert.NoError(t, checkReportFile(path), "an existing file is fine")

	err = checkReportFile(filepath.Join(t.TempDir(), "missing", "report.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report file")
}

func TestRunViewWritesJSONFile(t *testing.T) {
	r := sampleReport()
	r.Iterations = []benchmark.IterationResult{{Seq: 1, SandboxID: "sbx-1"}}
	path := filepath.Join(t.TempDir(), "report.json")

	require.NoError(t, newRunView("run-1", "system/base", 2, 300, r).writeJSONFile(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded), "the file holds one JSON document")
	assert.Equal(t, "run-1", decoded["run_id"])
	phases := decoded["phases"].([]any)
	require.Len(t, phases, 2)
	assert.InDelta(t, 1500, phases[0].(map[string]any)["avg_ms"], 1e-9, "the summary is in the file too")
	require.Len(t, decoded["iterations"].([]any), 1)

	// The document is the one --format json prints.
	var stdout []byte
	v := newRunView("run-1", "system/base", 2, 300, r)
	stdout, err = json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	assert.JSONEq(t, string(stdout), string(data))
}

func TestRunViewWriteJSONFileReportsBadPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "report.json")
	err := newRunView("run-1", "system/base", 1, 300, benchmark.Report{}).writeJSONFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write report")
}

func TestBreakdownFlag(t *testing.T) {
	op := parseLifecycle(t)
	assert.False(t, op.breakdown, "off by default")

	op = parseLifecycle(t, "--breakdown")
	assert.True(t, op.breakdown)
}

func TestLifecycleViewPrettyInterrupted(t *testing.T) {
	r := sampleReport()
	r.Interrupted = true
	r.Errors, r.Leaked = nil, nil

	var b strings.Builder
	require.NoError(t, newRunView("run-1", "system/base", 1, 300, r).writePretty(&b))
	out := b.String()

	assert.Contains(t, out, "Benchmark run-1 interrupted")
	assert.NotContains(t, out, "Errors:")
	assert.NotContains(t, out, "could not be killed")
}

func TestWithoutRetriesFailsRateLimitedRequestAtOnce(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	newClient := func(modifiers ...func(*client.Options)) *client.Client {
		opts := client.Options{APIKey: "test-key", APIURL: srv.URL}
		for _, modify := range modifiers {
			modify(&opts)
		}
		c, err := client.New(opts)
		require.NoError(t, err)
		return c
	}

	// The SDK's default retries the 429, which the benchmark must not time.
	_, err := newClient().Sandboxes().Kill(context.Background(), "sbx")
	require.Error(t, err)
	require.Greater(t, hits.Load(), int32(1), "the SDK retries a 429 by default")

	hits.Store(0)
	_, err = newClient(withoutRetries).Sandboxes().Kill(context.Background(), "sbx")
	require.Error(t, err)
	assert.EqualValues(t, 1, hits.Load())
}

// TestLifecycleTracesSDKConnections runs the benchmark against a fake control
// plane through the real SDK, checking that the requests it makes carry the
// benchmark's context, so their connection setup is measured.
func TestLifecycleTracesSDKConnections(t *testing.T) {
	var createBody atomic.Pointer[api.NewSandbox]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req api.NewSandbox
			_ = json.NewDecoder(r.Body).Decode(&req)
			createBody.Store(&req)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"sandboxID":"sbx-1","templateID":"tpl","clientID":"c","envdVersion":"0.2.0"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	sdk, err := client.New(client.Options{APIKey: "test-key", APIURL: srv.URL})
	require.NoError(t, err)

	op := parseLifecycle(t, "--skip-exec")
	report := benchmark.Run(context.Background(), newCreator(sdk.Sandboxes(), "run-1", op.template, op.timeout),
		benchmark.Options[*benchSandbox]{Count: 1, Concurrency: 1})

	require.Equal(t, 1, report.Succeeded, "errors: %v", report.Errors)
	for _, p := range report.Phases {
		assert.Equal(t, 1, p.Connect.Count, p.Phase)
	}
	// The first request dials a fresh connection, which takes measurable time.
	assert.Positive(t, report.Phases[0].Connect.Max)

	// Every sandbox is tagged with the run and marked as opened by the
	// benchmark.
	req := createBody.Load()
	require.NotNil(t, req)
	require.NotNil(t, req.Metadata)
	assert.Equal(t, "run-1", (*req.Metadata)[runIDMetadataKey])
	assert.Equal(t, "sandbox-bench", (*req.Metadata)[sandbox.ManageByMetadataKey])
}

func TestCreateFlags(t *testing.T) {
	op := &createOperation{}
	c := op.Command()

	require.NoError(t, c.ParseFlags([]string{"-t", "my-tpl", "-n", "50", "-c", "8", "--timeout", "600"}))
	assert.Equal(t, "my-tpl", op.template)
	assert.Equal(t, 50, op.count)
	assert.Equal(t, 8, op.concurrency)
	assert.EqualValues(t, 600, op.timeout)
	assert.NoError(t, op.validate())

	assert.Error(t, (&createOperation{}).Command().ParseFlags([]string{"--cmd", "ls"}),
		"create runs no command in its sandboxes")
}

func TestCreateDefaultTimeoutIsFiveMinutes(t *testing.T) {
	op := &createOperation{}
	require.NoError(t, op.Command().ParseFlags(nil))
	assert.EqualValues(t, 300, op.timeout)
}

func TestRunViewLeftRunning(t *testing.T) {
	r := sampleReport()
	r.Leaked = nil
	v := newRunView("run-1", "system/base", 1, 300, r)
	v.LeftRunning = 2

	var b strings.Builder
	require.NoError(t, v.writePretty(&b))
	out := b.String()
	assert.Contains(t, out, "2 sandbox(es) left running, reclaimed after their 5m0s timeout.")
	assert.Contains(t, out, "sandbox kill --all -m benchmark=run-1")

	data, err := json.Marshal(v)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"left_running":2`)
	assert.Contains(t, string(data), `"sandbox_timeout_sec":300`)
}

// TestCreateOnlyKillsNothing runs a create-only benchmark against a fake
// control plane through the real SDK.
func TestCreateOnlyKillsNothing(t *testing.T) {
	var creates, deletes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			creates.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"sandboxID":"sbx-1","templateID":"tpl","clientID":"c","envdVersion":"0.2.0"}`))
		case http.MethodDelete:
			deletes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	sdk, err := client.New(client.Options{APIKey: "test-key", APIURL: srv.URL})
	require.NoError(t, err)

	report := benchmark.Run(context.Background(),
		newCreator(sdk.Sandboxes(), "run-1", "tpl", 300),
		benchmark.Options[*benchSandbox]{Count: 3, Concurrency: 2, SkipKill: true})

	require.Equal(t, 3, report.Succeeded, "errors: %v", report.Errors)
	require.Len(t, report.Phases, 1)
	assert.Equal(t, benchmark.PhaseCreate, report.Phases[0].Phase)
	assert.EqualValues(t, 3, creates.Load())
	assert.Zero(t, deletes.Load())
}

// TestBenchSandboxKillNotFound pins the SDK contract benchSandbox.Kill relies
// on: only a 404 reads as not found, any other failure stays an error of its
// own.
func TestBenchSandboxKillNotFound(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		notFound bool
		wantErr  bool
	}{
		{"killed", http.StatusNoContent, false, false},
		{"not found", http.StatusNotFound, true, true},
		{"server error", http.StatusInternalServerError, false, true},
		{"conflict", http.StatusConflict, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"sandboxID":"sbx-1","templateID":"tpl","clientID":"c","envdVersion":"0.2.0"}`))
					return
				}
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			sdk, err := client.New(client.Options{APIKey: "test-key", APIURL: srv.URL})
			require.NoError(t, err)

			sbx, err := newCreator(sdk.Sandboxes(), "run-1", "tpl", 300)(context.Background())
			require.NoError(t, err)

			err = sbx.Kill(context.Background())
			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.notFound, errors.Is(err, benchmark.ErrSandboxNotFound))
		})
	}
}

func TestNewRunID(t *testing.T) {
	now := time.Date(2026, 11, 25, 11, 28, 42, 0, time.UTC)

	id := newRunID(now)
	assert.Regexp(t, `^20261125-112842-[0-9a-f]{6}$`, id)

	seen := map[string]bool{}
	for range 100 {
		seen[newRunID(now)] = true
	}
	assert.Greater(t, len(seen), 90, "runs started in the same second get different IDs")
}

func TestExitError(t *testing.T) {
	assert.NoError(t, exitError(benchmark.Report{Requested: 3, Started: 3, Succeeded: 3}))

	err := exitError(benchmark.Report{Requested: 3, Started: 3, Succeeded: 2, Failed: 1})
	require.Error(t, err)
	assert.Equal(t, "1 of 3 iteration(s) failed", err.Error())

	err = exitError(benchmark.Report{Requested: 100, Started: 3, Succeeded: 3, Interrupted: true})
	require.Error(t, err, "an interrupted run is not a success, even with no failure")
	assert.Equal(t, "interrupted: 3 of 100 iteration(s) started, 0 failed", err.Error())
}

func TestRunViewCleanupHint(t *testing.T) {
	render := func(r benchmark.Report, leftRunning int) string {
		v := newRunView("run-1", "system/base", 1, 300, r)
		v.LeftRunning = leftRunning
		var b strings.Builder
		require.NoError(t, v.writePretty(&b))
		return b.String()
	}
	const cleanup = "ucloud-sandbox-cli sandbox kill --all -m benchmark=run-1"

	clean := benchmark.Report{Phases: []benchmark.PhaseReport{{Phase: benchmark.PhaseCreate}}}
	assert.NotContains(t, render(clean, 0), cleanup, "nothing left behind, no hint")

	createFailed := benchmark.Report{Phases: []benchmark.PhaseReport{{Phase: benchmark.PhaseCreate, Failed: 2}}}
	out := render(createFailed, 0)
	assert.Contains(t, out, "2 create(s) failed; the service may still have started some of those sandboxes.")
	assert.Contains(t, out, "Clean them up with: "+cleanup)

	out = render(clean, 3)
	assert.Contains(t, out, "Kill them sooner with: "+cleanup)

	// Several reasons still print the command once.
	r := sampleReport()
	r.Phases[0].Failed = 1
	out = render(r, 2)
	assert.Contains(t, out, "could not be killed")
	assert.Contains(t, out, "Kill them with: ucloud-sandbox-cli sandbox kill sbx-1", "leaks are killed by ID, tagged or not")
	assert.Contains(t, out, "1 create(s) failed")
	assert.Contains(t, out, "2 sandbox(es) left running")
	assert.Equal(t, 1, strings.Count(out, cleanup))
	assert.Contains(t, out, "Clean them up with: "+cleanup)
}

func TestWriteHeader(t *testing.T) {
	op := &runOptions{template: "system/base", count: 10, concurrency: 2}

	var b strings.Builder
	op.writeHeader(&b, runSpec{name: "health"}, "20261125-225910-b2bbf0")

	assert.Equal(t, `Running health benchmark 20261125-225910-b2bbf0
  Template:     system/base
  Iterations:   10 (concurrency 2)
  Sandbox tag:  benchmark=20261125-225910-b2bbf0
  Cleanup:      ucloud-sandbox-cli sandbox kill --all -m benchmark=20261125-225910-b2bbf0

`, b.String())
}
