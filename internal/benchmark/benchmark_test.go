package benchmark

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeEmpty(t *testing.T) {
	assert.Equal(t, Stats{}, Summarize(nil))
}

func TestSummarizeSingleSample(t *testing.T) {
	s := Summarize([]time.Duration{5 * time.Millisecond})

	assert.Equal(t, 1, s.Count)
	for _, d := range []time.Duration{s.Avg, s.Min, s.P50, s.P95, s.P99, s.Max} {
		assert.Equal(t, 5*time.Millisecond, d)
	}
}

func TestSummarizeNearestRank(t *testing.T) {
	// 100..1 ms in reverse order, so sorting is exercised.
	samples := make([]time.Duration, 0, 100)
	for i := 100; i >= 1; i-- {
		samples = append(samples, time.Duration(i)*time.Millisecond)
	}

	s := Summarize(samples)

	assert.Equal(t, 100, s.Count)
	assert.Equal(t, 1*time.Millisecond, s.Min)
	assert.Equal(t, 100*time.Millisecond, s.Max)
	assert.Equal(t, 50500*time.Microsecond, s.Avg)
	assert.Equal(t, 50*time.Millisecond, s.P50)
	assert.Equal(t, 95*time.Millisecond, s.P95)
	assert.Equal(t, 99*time.Millisecond, s.P99)

	assert.Equal(t, 100*time.Millisecond, samples[0], "input is left unsorted")
}

func TestSummarizeSmallSetPercentiles(t *testing.T) {
	s := Summarize([]time.Duration{1, 2, 3})

	assert.Equal(t, time.Duration(2), s.P50)
	assert.Equal(t, time.Duration(3), s.P95)
	assert.Equal(t, time.Duration(3), s.P99)
}

// fakeSandbox is a Sandbox whose phases fail as configured.
type fakeSandbox struct {
	id      string
	execErr error
	killErr error

	execCalls atomic.Int32
	killCalls atomic.Int32
}

func (f *fakeSandbox) ID() string { return f.id }

func (f *fakeSandbox) Exec(context.Context) error {
	f.execCalls.Add(1)
	return f.execErr
}

func (f *fakeSandbox) Kill(context.Context) error {
	f.killCalls.Add(1)
	return f.killErr
}

// fakeService hands out fakeSandboxs and records them.
type fakeService struct {
	mu        sync.Mutex
	sandboxes []*fakeSandbox
	created   atomic.Int32

	// configure, when set, decides the behavior of the n-th (0-based) create.
	configure func(n int, sbx *fakeSandbox) error
}

func (s *fakeService) create(context.Context) (*fakeSandbox, error) {
	n := int(s.created.Add(1)) - 1
	sbx := &fakeSandbox{id: fmt.Sprintf("sbx-%03d", n)}

	if s.configure != nil {
		if err := s.configure(n, sbx); err != nil {
			return nil, err
		}
	}

	s.mu.Lock()
	s.sandboxes = append(s.sandboxes, sbx)
	s.mu.Unlock()

	return sbx, nil
}

// execer is a sandbox with an exec phase of its own.
type execer interface {
	Sandbox
	Exec(ctx context.Context) error
}

// execSteps returns the single exec step of a lifecycle run.
func execSteps[S execer]() []Step[S] {
	return []Step[S]{{
		Phase: PhaseExec,
		Run:   func(ctx context.Context, sbx S) error { return sbx.Exec(ctx) },
	}}
}

func phaseReport(t *testing.T, r Report, phase Phase) PhaseReport {
	t.Helper()
	for _, p := range r.Phases {
		if p.Phase == phase {
			return p
		}
	}
	require.Failf(t, "phase missing", "phase %q not in report", phase)
	return PhaseReport{}
}

func TestRunAllSucceed(t *testing.T) {
	svc := &fakeService{}

	var calls []Progress
	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{
		Steps:       execSteps[*fakeSandbox](),
		Count:       20,
		Concurrency: 4,
		OnIteration: func(p Progress) { calls = append(calls, p) },
	})

	assert.Equal(t, 20, report.Requested)
	assert.Equal(t, 20, report.Started)
	assert.Equal(t, 20, report.Succeeded)
	assert.Equal(t, 0, report.Failed)
	assert.False(t, report.Interrupted)
	assert.Empty(t, report.Errors)
	assert.Empty(t, report.Leaked)

	var phases []Phase
	for _, p := range report.Phases {
		phases = append(phases, p.Phase)
		assert.Equal(t, 20, p.Stats.Count, "phase %s", p.Phase)
		assert.Equal(t, 0, p.Failed, "phase %s", p.Phase)
	}
	assert.Equal(t, []Phase{PhaseCreate, PhaseExec, PhaseKill, PhaseTotal}, phases)

	for _, sbx := range svc.sandboxes {
		assert.EqualValues(t, 1, sbx.execCalls.Load())
		assert.EqualValues(t, 1, sbx.killCalls.Load())
	}

	require.Len(t, calls, 20)
	assert.Equal(t, Progress{Done: 20, Total: 20, Succeeded: 20}, calls[19])
}

func TestRunSkipExec(t *testing.T) {
	svc := &fakeService{}

	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{
		Count:       3,
		Concurrency: 1,
	})

	var phases []Phase
	for _, p := range report.Phases {
		phases = append(phases, p.Phase)
	}
	assert.Equal(t, []Phase{PhaseCreate, PhaseKill, PhaseTotal}, phases)

	for _, sbx := range svc.sandboxes {
		assert.EqualValues(t, 0, sbx.execCalls.Load())
		assert.EqualValues(t, 1, sbx.killCalls.Load())
	}
}

func TestRunFailures(t *testing.T) {
	svc := &fakeService{
		configure: func(n int, sbx *fakeSandbox) error {
			switch n % 5 {
			case 1:
				return errors.New("quota exceeded")
			case 2:
				sbx.execErr = errors.New("exit status 1")
			case 3:
				sbx.killErr = errors.New("kill timeout")
			case 4:
				sbx.killErr = fmt.Errorf("wrapped: %w", ErrSandboxNotFound)
			}
			return nil
		},
	}

	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{
		Steps:       execSteps[*fakeSandbox](),
		Count:       10,
		Concurrency: 1,
	})

	assert.Equal(t, 10, report.Started)
	assert.Equal(t, 2, report.Succeeded)
	assert.Equal(t, 8, report.Failed)

	create := phaseReport(t, report, PhaseCreate)
	assert.Equal(t, 8, create.Stats.Count)
	assert.Equal(t, 2, create.Failed)

	exec := phaseReport(t, report, PhaseExec)
	assert.Equal(t, 6, exec.Stats.Count)
	assert.Equal(t, 2, exec.Failed)

	kill := phaseReport(t, report, PhaseKill)
	assert.Equal(t, 4, kill.Stats.Count)
	assert.Equal(t, 4, kill.Failed)

	total := phaseReport(t, report, PhaseTotal)
	assert.Equal(t, 2, total.Stats.Count)
	assert.Equal(t, 8, total.Failed)

	// A failed exec still kills the sandbox.
	for _, sbx := range svc.sandboxes {
		assert.EqualValues(t, 1, sbx.killCalls.Load(), sbx.id)
	}

	// Only the sandbox whose kill failed outright is leaked; one that is
	// the service reports as not found is not.
	assert.Equal(t, []string{"sbx-003", "sbx-008"}, report.Leaked)

	assert.ElementsMatch(t, []ErrorCount{
		{Phase: PhaseCreate, Message: "quota exceeded", Count: 2},
		{Phase: PhaseExec, Message: "exit status 1", Count: 2},
		{Phase: PhaseKill, Message: "kill timeout", Count: 2},
		{Phase: PhaseKill, Message: "wrapped: sandbox not found (HTTP 404)", Count: 2},
	}, report.Errors)
}

func TestRunErrorsSortedByCount(t *testing.T) {
	svc := &fakeService{
		configure: func(n int, _ *fakeSandbox) error {
			if n == 0 {
				return errors.New("rare")
			}
			return errors.New("common")
		},
	}

	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{Steps: execSteps[*fakeSandbox](), Count: 4, Concurrency: 1})

	require.Len(t, report.Errors, 2)
	assert.Equal(t, ErrorCount{Phase: PhaseCreate, Message: "common", Count: 3}, report.Errors[0])
	assert.Equal(t, ErrorCount{Phase: PhaseCreate, Message: "rare", Count: 1}, report.Errors[1])
}

func TestRunRespectsConcurrency(t *testing.T) {
	var inFlight, peak atomic.Int32

	create := func(context.Context) (*fakeSandbox, error) {
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		inFlight.Add(-1)
		return &fakeSandbox{id: "sbx"}, nil
	}

	report := Run(context.Background(), create, Options[*fakeSandbox]{Steps: execSteps[*fakeSandbox](), Count: 30, Concurrency: 3})

	assert.Equal(t, 30, report.Succeeded)
	assert.LessOrEqual(t, peak.Load(), int32(3))
	assert.Greater(t, peak.Load(), int32(1), "iterations run concurrently")
}

func TestRunCancelStopsNewIterationsAndCleansUp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &fakeService{}
	release := make(chan struct{})

	create := func(c context.Context) (*fakeSandbox, error) {
		sbx, err := svc.create(c)
		if svc.created.Load() == 2 {
			// The second iteration is in flight when the run is cancelled.
			cancel()
			<-release
		}
		return sbx, err
	}

	go func() {
		<-ctx.Done()
		close(release)
	}()

	report := Run(ctx, create, Options[*fakeSandbox]{Steps: execSteps[*fakeSandbox](), Count: 100, Concurrency: 1})

	assert.True(t, report.Interrupted)
	assert.Equal(t, 100, report.Requested)
	assert.Equal(t, 2, report.Started)
	assert.Equal(t, 2, report.Succeeded, "the in-flight iteration completes on a detached context")
	assert.Len(t, report.Iterations, 2, "iterations that never started have no result")

	for _, sbx := range svc.sandboxes {
		assert.EqualValues(t, 1, sbx.killCalls.Load(), sbx.id)
	}
}

func TestRunZeroCount(t *testing.T) {
	svc := &fakeService{}

	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{Count: 0, Concurrency: 5})

	assert.Equal(t, 0, report.Started)
	assert.EqualValues(t, 0, svc.created.Load())
}

func TestReportRate(t *testing.T) {
	assert.InDelta(t, 2.5, Report{Succeeded: 5, Elapsed: 2 * time.Second}.Rate(), 1e-9)
	assert.Zero(t, Report{Succeeded: 5}.Rate())
}

// simulateConnect fires the httptrace hooks of ctx as an HTTP request taking d
// to obtain its connection would.
func simulateConnect(ctx context.Context, d time.Duration) {
	trace := httptrace.ContextClientTrace(ctx)
	if trace == nil {
		return
	}
	trace.GetConn("example.com:443")
	time.Sleep(d)
	trace.GotConn(httptrace.GotConnInfo{})
}

func TestConnTraceSumsSequentialRequests(t *testing.T) {
	ctx, trace := withConnTrace(context.Background())

	simulateConnect(ctx, 5*time.Millisecond)
	simulateConnect(ctx, 5*time.Millisecond)

	assert.GreaterOrEqual(t, trace.Total(), 10*time.Millisecond)
}

func TestConnTraceIgnoresUnpairedGotConn(t *testing.T) {
	ctx, trace := withConnTrace(context.Background())

	httptrace.ContextClientTrace(ctx).GotConn(httptrace.GotConnInfo{})

	assert.Zero(t, trace.Total())
}

// connectingSandbox spends connect obtaining a connection, then work
// responding, in every phase.
type connectingSandbox struct {
	connect, work time.Duration
}

func (i connectingSandbox) phase(ctx context.Context) error {
	simulateConnect(ctx, i.connect)
	time.Sleep(i.work)
	return nil
}

func (i connectingSandbox) ID() string                     { return "sbx" }
func (i connectingSandbox) Exec(ctx context.Context) error { return i.phase(ctx) }
func (i connectingSandbox) Kill(ctx context.Context) error { return i.phase(ctx) }

func TestRunSplitsConnectFromService(t *testing.T) {
	const connect, work = 20 * time.Millisecond, 10 * time.Millisecond
	sbx := connectingSandbox{connect: connect, work: work}

	create := func(ctx context.Context) (connectingSandbox, error) {
		return sbx, sbx.phase(ctx)
	}

	report := Run(context.Background(), create, Options[connectingSandbox]{
		Steps: execSteps[connectingSandbox](), Count: 2, Concurrency: 1,
	})
	require.Equal(t, 2, report.Succeeded)

	for _, phase := range []Phase{PhaseCreate, PhaseExec, PhaseKill} {
		p := phaseReport(t, report, phase)
		assert.Equal(t, 2, p.Connect.Count, phase)
		assert.GreaterOrEqual(t, p.Connect.Min, connect, phase)
		assert.GreaterOrEqual(t, p.Service.Min, work, phase)
		assert.Less(t, p.Service.Max, connect, "%s: connection setup is not counted as service time", phase)
	}

	total := phaseReport(t, report, PhaseTotal)
	assert.GreaterOrEqual(t, total.Connect.Min, 3*connect, "total sums the connection setup of every phase")
	assert.GreaterOrEqual(t, total.Service.Min, 3*work)
}

func TestRunFailedPhaseRecordsNoConnect(t *testing.T) {
	svc := &fakeService{
		configure: func(_ int, sbx *fakeSandbox) error {
			sbx.execErr = errors.New("boom")
			return nil
		},
	}

	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{Steps: execSteps[*fakeSandbox](), Count: 1, Concurrency: 1})

	exec := phaseReport(t, report, PhaseExec)
	assert.Zero(t, exec.Connect.Count)
	assert.Zero(t, exec.Service.Count)
	assert.Equal(t, 1, phaseReport(t, report, PhaseCreate).Connect.Count)
}

func TestRunCreateOnly(t *testing.T) {
	svc := &fakeService{
		configure: func(n int, _ *fakeSandbox) error {
			if n == 0 {
				return errors.New("quota exceeded")
			}
			return nil
		},
	}

	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{
		Count:       4,
		Concurrency: 2,
		SkipKill:    true,
	})

	require.Len(t, report.Phases, 1, "a lone phase has no total")
	create := report.Phases[0]
	assert.Equal(t, PhaseCreate, create.Phase)
	assert.Equal(t, 3, create.Stats.Count)
	assert.Equal(t, 1, create.Failed)

	assert.Equal(t, 3, report.Succeeded)
	assert.Equal(t, 1, report.Failed)
	assert.Empty(t, report.Leaked, "sandboxes left running on purpose are not leaks")

	require.Len(t, svc.sandboxes, 3)
	for _, sbx := range svc.sandboxes {
		assert.EqualValues(t, 0, sbx.execCalls.Load(), sbx.id)
		assert.EqualValues(t, 0, sbx.killCalls.Load(), sbx.id)
	}
}

func TestOptionsPhases(t *testing.T) {
	exec := execSteps[*fakeSandbox]()
	type opts = Options[*fakeSandbox]

	assert.Equal(t, []Phase{PhaseCreate, PhaseExec, PhaseKill, PhaseTotal}, opts{Steps: exec}.phases())
	assert.Equal(t, []Phase{PhaseCreate, PhaseKill, PhaseTotal}, opts{}.phases())
	assert.Equal(t, []Phase{PhaseCreate, PhaseExec, PhaseTotal}, opts{Steps: exec, SkipKill: true}.phases())
	assert.Equal(t, []Phase{PhaseCreate}, opts{SkipKill: true}.phases())

	steps := []Step[*fakeSandbox]{{Phase: "a"}, {Phase: "b"}}
	assert.Equal(t, []Phase{PhaseCreate, "a", "b", PhaseKill, PhaseTotal}, opts{Steps: steps}.phases())
}

func TestNormalizeMessage(t *testing.T) {
	assert.Equal(t,
		"command exited with code 127: bash: nope: command not found",
		normalizeMessage("command exited with code 127: bash: nope: command not found\n", "sbx-1"))
	assert.Equal(t,
		"line one line two",
		normalizeMessage("  line one\n\tline two\r\n", ""))
	assert.Equal(t,
		`Post "https://49983-<sandbox>.example.com/process.Process/Start": dial tcp: i/o timeout`,
		normalizeMessage(`Post "https://49983-ibx7k2.example.com/process.Process/Start": dial tcp: i/o timeout`, "ibx7k2"))
}

func TestRunGroupsErrorsAcrossSandboxes(t *testing.T) {
	svc := &fakeService{
		configure: func(n int, sbx *fakeSandbox) error {
			// Every exec fails with a message naming its own sandbox and
			// spanning several lines.
			sbx.execErr = fmt.Errorf("dial https://49983-%s.example.com: refused\nretry later\n", sbx.id)
			return nil
		},
	}

	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{Steps: execSteps[*fakeSandbox](), Count: 5, Concurrency: 2})

	require.Len(t, report.Errors, 1)
	assert.Equal(t, ErrorCount{
		Phase:   PhaseExec,
		Message: "dial https://49983-<sandbox>.example.com: refused retry later",
		Count:   5,
	}, report.Errors[0])
}

func TestRunStepsRunInOrderPastFailures(t *testing.T) {
	var mu sync.Mutex
	var order []Phase
	step := func(phase Phase, err error) Step[*fakeSandbox] {
		return Step[*fakeSandbox]{Phase: phase, Run: func(context.Context, *fakeSandbox) error {
			mu.Lock()
			order = append(order, phase)
			mu.Unlock()
			return err
		}}
	}

	svc := &fakeService{}
	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{
		Count:       1,
		Concurrency: 1,
		Steps: []Step[*fakeSandbox]{
			step("internet", errors.New("could not resolve host")),
			step("filesystem", nil),
		},
	})

	assert.Equal(t, []Phase{"internet", "filesystem"}, order, "a failed step does not stop the next")
	assert.Equal(t, 1, phaseReport(t, report, "internet").Failed)
	assert.Equal(t, 1, phaseReport(t, report, "filesystem").Stats.Count)
	assert.EqualValues(t, 1, svc.sandboxes[0].killCalls.Load())
}

func TestRunUntimedStep(t *testing.T) {
	var order []Phase
	record := func(phase Phase, sleep time.Duration, err error) Step[*fakeSandbox] {
		return Step[*fakeSandbox]{Phase: phase, Run: func(context.Context, *fakeSandbox) error {
			order = append(order, phase)
			time.Sleep(sleep)
			return err
		}}
	}
	cleanup := record("cleanup", 20*time.Millisecond, nil)
	cleanup.Untimed = true

	svc := &fakeService{}
	opts := Options[*fakeSandbox]{
		Count:       1,
		Concurrency: 1,
		Steps:       []Step[*fakeSandbox]{record("work", 0, nil), cleanup, record("after", 0, nil)},
	}
	report := Run(context.Background(), svc.create, opts)

	assert.Equal(t, []Phase{"work", "cleanup", "after"}, order, "untimed steps run in their place")
	assert.Equal(t, []Phase{PhaseCreate, "work", "after", PhaseKill, PhaseTotal}, opts.phases(), "and get no row")
	for _, p := range report.Phases {
		assert.NotEqual(t, Phase("cleanup"), p.Phase)
	}
	assert.Equal(t, 1, report.Succeeded)
	assert.Less(t, phaseReport(t, report, "after").Stats.Max, 20*time.Millisecond)
}

func TestRunUntimedStepFailureFailsIterationAndLeaks(t *testing.T) {
	svc := &fakeService{}
	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{
		Count:       1,
		Concurrency: 1,
		Steps: []Step[*fakeSandbox]{{
			Phase:   "cleanup",
			Untimed: true,
			Run: func(context.Context, *fakeSandbox) error {
				return &LeakedError{IDs: []string{"sbx-fork"}, Err: errors.New("HTTP 500")}
			},
		}},
	})

	assert.Equal(t, 0, report.Succeeded, "a leak is a failure")
	assert.Equal(t, 1, report.Failed)
	assert.Equal(t, []string{"sbx-fork"}, report.Leaked)
	assert.Equal(t, []ErrorCount{{Phase: "cleanup", Message: "could not kill 1 sandbox(es): HTTP 500", Count: 1}}, report.Errors)
	assert.EqualValues(t, 1, svc.sandboxes[0].killCalls.Load(), "the sandbox is still killed")
}

func TestRunLeaksBothStepAndKillFailures(t *testing.T) {
	svc := &fakeService{configure: func(_ int, sbx *fakeSandbox) error {
		sbx.killErr = errors.New("kill timeout")
		return nil
	}}

	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{
		Count:       1,
		Concurrency: 1,
		Steps: []Step[*fakeSandbox]{{Phase: "cleanup", Untimed: true, Run: func(context.Context, *fakeSandbox) error {
			return &LeakedError{IDs: []string{"sbx-fork"}, Err: errors.New("HTTP 500")}
		}}},
	})

	assert.Equal(t, []string{"sbx-000", "sbx-fork"}, report.Leaked)
}

// phaseNames returns the phases of an iteration in the order they ran, each
// suffixed with "!" when it failed.
func phaseNames(r IterationResult) []string {
	names := make([]string, 0, len(r.Phases))
	for _, p := range r.Phases {
		if p.Err != nil {
			names = append(names, string(p.Phase)+"!")
		} else {
			names = append(names, string(p.Phase))
		}
	}
	return names
}

func TestRunRecordsIterations(t *testing.T) {
	svc := &fakeService{
		configure: func(n int, sbx *fakeSandbox) error {
			switch n {
			case 1:
				return errors.New("quota exceeded")
			case 2:
				sbx.execErr = errors.New("exit status 1")
			case 3:
				sbx.killErr = errors.New("kill timeout")
			}
			return nil
		},
	}

	report := Run(context.Background(), svc.create, Options[*fakeSandbox]{
		Steps:       execSteps[*fakeSandbox](),
		Count:       4,
		Concurrency: 1,
	})

	require.Len(t, report.Iterations, 4)

	var prevStart time.Duration
	for i, it := range report.Iterations {
		assert.Equal(t, i+1, it.Seq)
		assert.GreaterOrEqual(t, it.Start, prevStart, "a serial run starts its iterations in order")
		prevStart = it.Start
	}

	ok := report.Iterations[0]
	assert.True(t, ok.Succeeded())
	assert.Equal(t, "sbx-000", ok.SandboxID)
	assert.Equal(t, []string{"create", "exec", "kill", "total"}, phaseNames(ok))
	total := ok.Phases[3]
	assert.GreaterOrEqual(t, total.Duration, ok.Phases[0].Duration+ok.Phases[1].Duration+ok.Phases[2].Duration)

	createFailed := report.Iterations[1]
	assert.False(t, createFailed.Succeeded())
	assert.Empty(t, createFailed.SandboxID)
	assert.Equal(t, []string{"create!"}, phaseNames(createFailed), "no phase runs after a failed create")
	assert.EqualError(t, createFailed.Phases[0].Err, "quota exceeded")
	assert.Zero(t, createFailed.Phases[0].Duration)

	execFailed := report.Iterations[2]
	assert.Equal(t, []string{"create", "exec!", "kill"}, phaseNames(execFailed), "a failed iteration has no total")

	killFailed := report.Iterations[3]
	assert.Equal(t, []string{"create", "exec", "kill!"}, phaseNames(killFailed))
	assert.Equal(t, "sbx-003", killFailed.SandboxID)
	assert.Equal(t, []string{"sbx-003"}, killFailed.Leaked)
}

func TestRunOrdersIterationsBySeq(t *testing.T) {
	var n atomic.Int32
	create := func(context.Context) (*fakeSandbox, error) {
		// Later iterations finish first, so completion order is not seq order.
		time.Sleep(time.Duration(10-n.Add(1)%10) * time.Millisecond)
		return &fakeSandbox{id: "sbx"}, nil
	}

	report := Run(context.Background(), create, Options[*fakeSandbox]{Count: 20, Concurrency: 4})

	require.Len(t, report.Iterations, 20)
	for i, it := range report.Iterations {
		assert.Equal(t, i+1, it.Seq)
	}
}

func TestRunRecordsUntimedStepOnlyWhenItFails(t *testing.T) {
	step := func(phase Phase, untimed bool, err error) Step[*fakeSandbox] {
		return Step[*fakeSandbox]{Phase: phase, Untimed: untimed, Run: func(context.Context, *fakeSandbox) error {
			return err
		}}
	}

	run := func(cleanupErr error) IterationResult {
		svc := &fakeService{}
		report := Run(context.Background(), svc.create, Options[*fakeSandbox]{
			Count:       1,
			Concurrency: 1,
			Steps: []Step[*fakeSandbox]{
				step("work", false, nil),
				step("cleanup", true, cleanupErr),
				step("after", false, nil),
			},
		})
		require.Len(t, report.Iterations, 1)
		return report.Iterations[0]
	}

	assert.Equal(t, []string{"create", "work", "after", "kill", "total"}, phaseNames(run(nil)))
	assert.Equal(t, []string{"create", "work", "cleanup!", "after", "kill"}, phaseNames(run(errors.New("HTTP 500"))),
		"a failed untimed step shows up where it ran")
}
