// Package benchmark measures the latency and health of sandbox operations
// under load.
//
// The runner is decoupled from the SDK: it drives Sandbox values handed out
// by a CreateFunc through a list of Steps, so the scheduling and statistics
// can be tested without a real sandbox service.
package benchmark

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Phase names one timed step of an iteration.
type Phase string

const (
	PhaseCreate Phase = "create"
	PhaseExec   Phase = "exec"
	PhaseKill   Phase = "kill"

	// PhaseTotal spans a whole iteration and is only recorded for iterations
	// in which every phase succeeded.
	PhaseTotal Phase = "total"
)

// ErrSandboxNotFound is returned by Sandbox.Kill when the service answers
// that the sandbox does not exist (HTTP 404). The kill counts as failed, but
// the sandbox is not reported as leaked.
var ErrSandboxNotFound = errors.New("sandbox not found (HTTP 404)")

// Sandbox is the sandbox an iteration creates and works on.
type Sandbox interface {
	ID() string
	Kill(ctx context.Context) error
}

// CreateFunc creates the sandbox one iteration works on.
type CreateFunc[S Sandbox] func(ctx context.Context) (S, error)

// Step is one timed operation run on the sandbox between create and kill.
type Step[S Sandbox] struct {
	Phase Phase
	Run   func(ctx context.Context, sbx S) error
}

// Options shape a benchmark run.
type Options[S Sandbox] struct {
	// Count is the number of iterations to run.
	Count int

	// Concurrency is the number of iterations in flight at once.
	Concurrency int

	// Steps run in order on every sandbox after it is created. A failing step
	// does not stop the ones after it, so every step reports on its own.
	Steps []Step[S]

	// SkipKill leaves out the kill phase: every sandbox created is left running
	// until its own timeout reclaims it.
	SkipKill bool

	// OnIteration, when set, is called after every finished iteration with the
	// running totals. Calls are serialized.
	OnIteration func(progress Progress)
}

// phases lists the timed phases of an iteration in execution order, followed
// by PhaseTotal when there is more than one to add up.
func (o Options[S]) phases() []Phase {
	phases := []Phase{PhaseCreate}
	for _, step := range o.Steps {
		phases = append(phases, step.Phase)
	}
	if !o.SkipKill {
		phases = append(phases, PhaseKill)
	}
	if len(phases) > 1 {
		phases = append(phases, PhaseTotal)
	}
	return phases
}

// Progress is the running state of a benchmark, as reported to OnIteration.
type Progress struct {
	Done      int
	Total     int
	Succeeded int
	Failed    int
}

// PhaseError is the failure of one phase of an iteration.
type PhaseError struct {
	Phase Phase
	Err   error
}

// IterationResult is the outcome of one iteration.
type IterationResult struct {
	SandboxID string

	// Durations holds the time taken by every phase that succeeded.
	Durations map[Phase]time.Duration

	// Connects holds, for every phase in Durations, the part of its time spent
	// obtaining HTTP connections (DNS, dial, TLS). The rest is the time the
	// service took to respond.
	Connects map[Phase]time.Duration

	// Errors lists the phases that failed, in the order they ran.
	Errors []PhaseError

	// Leaked reports that the sandbox was created but could not be killed.
	Leaked bool
}

// Succeeded reports whether every phase of the iteration succeeded.
func (r IterationResult) Succeeded() bool {
	return len(r.Errors) == 0
}

// Run runs opts.Count iterations with at most opts.Concurrency of them in
// flight. An iteration creates a sandbox, runs opts.Steps on it and kills it.
//
// Cancelling ctx stops new iterations from starting. Iterations already in
// flight run to completion on a context detached from ctx, so every sandbox
// they created is still killed (unless SkipKill is set).
func Run[S Sandbox](ctx context.Context, create CreateFunc[S], opts Options[S]) Report {
	count := max(opts.Count, 0)
	concurrency := min(max(opts.Concurrency, 1), max(count, 1))

	// In-flight iterations must be able to clean up after a cancellation.
	iterCtx := context.WithoutCancel(ctx)

	jobs := make(chan struct{})
	var (
		mu       sync.Mutex
		results  = make([]IterationResult, 0, count)
		progress = Progress{Total: count}
		wg       sync.WaitGroup
	)

	start := time.Now()

	for range concurrency {
		wg.Go(func() {
			for range jobs {
				// The dispatcher may hand out a job in the same instant ctx is
				// cancelled; drop it rather than start a new sandbox.
				if ctx.Err() != nil {
					continue
				}

				result := runIteration(iterCtx, create, opts)

				mu.Lock()
				results = append(results, result)
				progress.Done++
				if result.Succeeded() {
					progress.Succeeded++
				} else {
					progress.Failed++
				}
				if opts.OnIteration != nil {
					opts.OnIteration(progress)
				}
				mu.Unlock()
			}
		})
	}

dispatch:
	for range count {
		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- struct{}{}:
		}
	}
	close(jobs)
	wg.Wait()

	// A cancellation that arrives after the last iteration started cut nothing
	// short, so only iterations that never ran mark the run as interrupted.
	interrupted := len(results) < count

	return buildReport(results, count, opts.phases(), interrupted, time.Since(start))
}

// runIteration creates a sandbox, runs every step on it and kills it. The
// sandbox is killed even when a step fails.
func runIteration[S Sandbox](ctx context.Context, create CreateFunc[S], opts Options[S]) IterationResult {
	result := IterationResult{
		Durations: make(map[Phase]time.Duration, len(opts.Steps)+3),
		Connects:  make(map[Phase]time.Duration, len(opts.Steps)+3),
	}

	iterStart := time.Now()

	var sbx S
	err := result.timePhase(ctx, PhaseCreate, func(ctx context.Context) error {
		var err error
		sbx, err = create(ctx)
		return err
	})
	if err != nil {
		return result
	}
	result.SandboxID = sbx.ID()

	for _, step := range opts.Steps {
		result.timePhase(ctx, step.Phase, func(ctx context.Context) error {
			return step.Run(ctx, sbx)
		})
	}

	if !opts.SkipKill {
		killErr := result.timePhase(ctx, PhaseKill, sbx.Kill)
		result.Leaked = killErr != nil && !errors.Is(killErr, ErrSandboxNotFound)
	}

	if result.Succeeded() {
		var connect time.Duration
		for _, d := range result.Connects {
			connect += d
		}
		result.Durations[PhaseTotal] = time.Since(iterStart)
		result.Connects[PhaseTotal] = connect
	}

	return result
}

// timePhase runs one phase, recording its duration and connection setup time
// when it succeeds and its error when it fails.
func (r *IterationResult) timePhase(ctx context.Context, phase Phase, run func(context.Context) error) error {
	traced, conn := withConnTrace(ctx)

	start := time.Now()
	if err := run(traced); err != nil {
		r.Errors = append(r.Errors, PhaseError{Phase: phase, Err: err})
		return err
	}
	r.Durations[phase] = time.Since(start)
	r.Connects[phase] = conn.Total()

	return nil
}
