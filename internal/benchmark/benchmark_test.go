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

// fakeInstance is an Instance whose phases fail as configured.
type fakeInstance struct {
	id      string
	execErr error
	killErr error

	execCalls atomic.Int32
	killCalls atomic.Int32
}

func (f *fakeInstance) ID() string { return f.id }

func (f *fakeInstance) Exec(context.Context) error {
	f.execCalls.Add(1)
	return f.execErr
}

func (f *fakeInstance) Kill(context.Context) error {
	f.killCalls.Add(1)
	return f.killErr
}

// fakeService hands out fakeInstances and records them.
type fakeService struct {
	mu        sync.Mutex
	instances []*fakeInstance
	created   atomic.Int32

	// configure, when set, decides the behavior of the n-th (0-based) create.
	configure func(n int, inst *fakeInstance) error
}

func (s *fakeService) create(context.Context) (Instance, error) {
	n := int(s.created.Add(1)) - 1
	inst := &fakeInstance{id: fmt.Sprintf("sbx-%03d", n)}

	if s.configure != nil {
		if err := s.configure(n, inst); err != nil {
			return nil, err
		}
	}

	s.mu.Lock()
	s.instances = append(s.instances, inst)
	s.mu.Unlock()

	return inst, nil
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

func TestRunLifecycleAllSucceed(t *testing.T) {
	svc := &fakeService{}

	var calls []Progress
	report := RunLifecycle(context.Background(), svc.create, LifecycleOptions{
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

	for _, inst := range svc.instances {
		assert.EqualValues(t, 1, inst.execCalls.Load())
		assert.EqualValues(t, 1, inst.killCalls.Load())
	}

	require.Len(t, calls, 20)
	assert.Equal(t, Progress{Done: 20, Total: 20, Succeeded: 20}, calls[19])
}

func TestRunLifecycleSkipExec(t *testing.T) {
	svc := &fakeService{}

	report := RunLifecycle(context.Background(), svc.create, LifecycleOptions{
		Count:       3,
		Concurrency: 1,
		SkipExec:    true,
	})

	var phases []Phase
	for _, p := range report.Phases {
		phases = append(phases, p.Phase)
	}
	assert.Equal(t, []Phase{PhaseCreate, PhaseKill, PhaseTotal}, phases)

	for _, inst := range svc.instances {
		assert.EqualValues(t, 0, inst.execCalls.Load())
		assert.EqualValues(t, 1, inst.killCalls.Load())
	}
}

func TestRunLifecycleFailures(t *testing.T) {
	svc := &fakeService{
		configure: func(n int, inst *fakeInstance) error {
			switch n % 5 {
			case 1:
				return errors.New("quota exceeded")
			case 2:
				inst.execErr = errors.New("exit status 1")
			case 3:
				inst.killErr = errors.New("kill timeout")
			case 4:
				inst.killErr = fmt.Errorf("wrapped: %w", ErrSandboxNotFound)
			}
			return nil
		},
	}

	report := RunLifecycle(context.Background(), svc.create, LifecycleOptions{
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
	for _, inst := range svc.instances {
		assert.EqualValues(t, 1, inst.killCalls.Load(), inst.id)
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

func TestRunLifecycleErrorsSortedByCount(t *testing.T) {
	svc := &fakeService{
		configure: func(n int, _ *fakeInstance) error {
			if n == 0 {
				return errors.New("rare")
			}
			return errors.New("common")
		},
	}

	report := RunLifecycle(context.Background(), svc.create, LifecycleOptions{Count: 4, Concurrency: 1})

	require.Len(t, report.Errors, 2)
	assert.Equal(t, ErrorCount{Phase: PhaseCreate, Message: "common", Count: 3}, report.Errors[0])
	assert.Equal(t, ErrorCount{Phase: PhaseCreate, Message: "rare", Count: 1}, report.Errors[1])
}

func TestRunLifecycleRespectsConcurrency(t *testing.T) {
	var inFlight, peak atomic.Int32

	create := func(context.Context) (Instance, error) {
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		inFlight.Add(-1)
		return &fakeInstance{id: "sbx"}, nil
	}

	report := RunLifecycle(context.Background(), create, LifecycleOptions{Count: 30, Concurrency: 3})

	assert.Equal(t, 30, report.Succeeded)
	assert.LessOrEqual(t, peak.Load(), int32(3))
	assert.Greater(t, peak.Load(), int32(1), "iterations run concurrently")
}

func TestRunLifecycleCancelStopsNewIterationsAndCleansUp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &fakeService{}
	release := make(chan struct{})

	create := func(c context.Context) (Instance, error) {
		inst, err := svc.create(c)
		if svc.created.Load() == 2 {
			// The second iteration is in flight when the run is cancelled.
			cancel()
			<-release
		}
		return inst, err
	}

	go func() {
		<-ctx.Done()
		close(release)
	}()

	report := RunLifecycle(ctx, create, LifecycleOptions{Count: 100, Concurrency: 1})

	assert.True(t, report.Interrupted)
	assert.Equal(t, 100, report.Requested)
	assert.Equal(t, 2, report.Started)
	assert.Equal(t, 2, report.Succeeded, "the in-flight iteration completes on a detached context")

	for _, inst := range svc.instances {
		assert.EqualValues(t, 1, inst.killCalls.Load(), inst.id)
	}
}

func TestRunLifecycleZeroCount(t *testing.T) {
	svc := &fakeService{}

	report := RunLifecycle(context.Background(), svc.create, LifecycleOptions{Count: 0, Concurrency: 5})

	assert.Equal(t, 0, report.Started)
	assert.EqualValues(t, 0, svc.created.Load())
}

func TestReportThroughput(t *testing.T) {
	assert.InDelta(t, 2.5, Report{Succeeded: 5, Elapsed: 2 * time.Second}.Throughput(), 1e-9)
	assert.Zero(t, Report{Succeeded: 5}.Throughput())
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

// connectingInstance spends connect obtaining a connection, then work
// responding, in every phase.
type connectingInstance struct {
	connect, work time.Duration
}

func (i connectingInstance) phase(ctx context.Context) error {
	simulateConnect(ctx, i.connect)
	time.Sleep(i.work)
	return nil
}

func (i connectingInstance) ID() string                     { return "sbx" }
func (i connectingInstance) Exec(ctx context.Context) error { return i.phase(ctx) }
func (i connectingInstance) Kill(ctx context.Context) error { return i.phase(ctx) }

func TestRunLifecycleSplitsConnectFromService(t *testing.T) {
	const connect, work = 20 * time.Millisecond, 10 * time.Millisecond
	inst := connectingInstance{connect: connect, work: work}

	create := func(ctx context.Context) (Instance, error) {
		return inst, inst.phase(ctx)
	}

	report := RunLifecycle(context.Background(), create, LifecycleOptions{Count: 2, Concurrency: 1})
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

func TestRunLifecycleFailedPhaseRecordsNoConnect(t *testing.T) {
	svc := &fakeService{
		configure: func(_ int, inst *fakeInstance) error {
			inst.execErr = errors.New("boom")
			return nil
		},
	}

	report := RunLifecycle(context.Background(), svc.create, LifecycleOptions{Count: 1, Concurrency: 1})

	exec := phaseReport(t, report, PhaseExec)
	assert.Zero(t, exec.Connect.Count)
	assert.Zero(t, exec.Service.Count)
	assert.Equal(t, 1, phaseReport(t, report, PhaseCreate).Connect.Count)
}

func TestRunLifecycleCreateOnly(t *testing.T) {
	svc := &fakeService{
		configure: func(n int, _ *fakeInstance) error {
			if n == 0 {
				return errors.New("quota exceeded")
			}
			return nil
		},
	}

	report := RunLifecycle(context.Background(), svc.create, LifecycleOptions{
		Count:       4,
		Concurrency: 2,
		SkipExec:    true,
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

	require.Len(t, svc.instances, 3)
	for _, inst := range svc.instances {
		assert.EqualValues(t, 0, inst.execCalls.Load(), inst.id)
		assert.EqualValues(t, 0, inst.killCalls.Load(), inst.id)
	}
}

func TestLifecycleOptionsPhases(t *testing.T) {
	assert.Equal(t, []Phase{PhaseCreate, PhaseExec, PhaseKill, PhaseTotal}, LifecycleOptions{}.phases())
	assert.Equal(t, []Phase{PhaseCreate, PhaseKill, PhaseTotal}, LifecycleOptions{SkipExec: true}.phases())
	assert.Equal(t, []Phase{PhaseCreate, PhaseExec, PhaseTotal}, LifecycleOptions{SkipKill: true}.phases())
	assert.Equal(t, []Phase{PhaseCreate}, LifecycleOptions{SkipExec: true, SkipKill: true}.phases())
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

func TestRunLifecycleGroupsErrorsAcrossSandboxes(t *testing.T) {
	svc := &fakeService{
		configure: func(n int, inst *fakeInstance) error {
			// Every exec fails with a message naming its own sandbox and
			// spanning several lines.
			inst.execErr = fmt.Errorf("dial https://49983-%s.example.com: refused\nretry later\n", inst.id)
			return nil
		},
	}

	report := RunLifecycle(context.Background(), svc.create, LifecycleOptions{Count: 5, Concurrency: 2})

	require.Len(t, report.Errors, 1)
	assert.Equal(t, ErrorCount{
		Phase:   PhaseExec,
		Message: "dial https://49983-<sandbox>.example.com: refused retry later",
		Count:   5,
	}, report.Errors[0])
}
