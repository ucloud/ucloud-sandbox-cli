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
	"fmt"
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

// Step is one operation run on the sandbox between create and kill.
type Step[S Sandbox] struct {
	Phase Phase
	Run   func(ctx context.Context, sbx S) error

	// Untimed steps do housekeeping, such as removing what earlier steps
	// created. They fail the iteration like any step, but record no duration
	// and get no row of their own in the report.
	Untimed bool
}

// LeakedError is returned by a step that could not kill sandboxes it was
// responsible for. Their IDs are reported as leaked.
type LeakedError struct {
	IDs []string
	Err error
}

func (e *LeakedError) Error() string {
	return fmt.Sprintf("could not kill %d sandbox(es): %v", len(e.IDs), e.Err)
}

func (e *LeakedError) Unwrap() error { return e.Err }

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
		if !step.Untimed {
			phases = append(phases, step.Phase)
		}
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

// PhaseResult is the outcome of one phase of an iteration.
type PhaseResult struct {
	Phase Phase

	// Duration is the time the phase took, and Connect the part of it spent
	// obtaining HTTP connections (DNS, dial, TLS); the rest is the time the
	// service took to respond. Both are zero when the phase failed.
	Duration time.Duration
	Connect  time.Duration

	// Err is the error the phase failed with, nil when it succeeded.
	Err error
}

// IterationResult is the outcome of one iteration.
type IterationResult struct {
	// Seq numbers the iterations from 1 in the order they were handed out.
	Seq int

	// Start is when the iteration started, as an offset from the start of
	// the run.
	Start time.Duration

	SandboxID string

	// Phases lists the phases the iteration ran, in the order it ran them,
	// followed by PhaseTotal when every phase succeeded. A successful untimed
	// step leaves no entry; a phase never reached, after a failed create say,
	// leaves none either.
	Phases []PhaseResult

	// Leaked lists the sandboxes the iteration created but could not kill.
	Leaked []string
}

// Succeeded reports whether every phase of the iteration succeeded.
func (r IterationResult) Succeeded() bool {
	for _, p := range r.Phases {
		if p.Err != nil {
			return false
		}
	}
	return true
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

	jobs := make(chan int)
	var (
		mu       sync.Mutex
		results  = make([]IterationResult, 0, count)
		progress = Progress{Total: count}
		wg       sync.WaitGroup
	)

	start := time.Now()

	for range concurrency {
		wg.Go(func() {
			for seq := range jobs {
				// The dispatcher may hand out a job in the same instant ctx is
				// cancelled; drop it rather than start a new sandbox.
				if ctx.Err() != nil {
					continue
				}

				result := runIteration(iterCtx, create, opts, seq, start)

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
	for seq := 1; seq <= count; seq++ {
		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- seq:
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
// sandbox is killed even when a step fails. seq numbers the iteration, and
// runStart is when the run started.
func runIteration[S Sandbox](ctx context.Context, create CreateFunc[S], opts Options[S], seq int, runStart time.Time) IterationResult {
	iterStart := time.Now()

	result := IterationResult{
		Seq:    seq,
		Start:  iterStart.Sub(runStart),
		Phases: make([]PhaseResult, 0, len(opts.Steps)+3),
	}

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
		run := func(ctx context.Context) error { return step.Run(ctx, sbx) }
		if step.Untimed {
			result.runUntimed(ctx, step.Phase, run)
		} else {
			result.timePhase(ctx, step.Phase, run)
		}
	}

	if !opts.SkipKill {
		killErr := result.timePhase(ctx, PhaseKill, sbx.Kill)
		if killErr != nil && !errors.Is(killErr, ErrSandboxNotFound) {
			result.Leaked = append(result.Leaked, result.SandboxID)
		}
	}

	if result.Succeeded() {
		var connect time.Duration
		for _, p := range result.Phases {
			connect += p.Connect
		}
		result.Phases = append(result.Phases, PhaseResult{
			Phase:    PhaseTotal,
			Duration: time.Since(iterStart),
			Connect:  connect,
		})
	}

	return result
}

// timePhase runs one phase, recording its duration and connection setup time
// when it succeeds and its error when it fails.
func (r *IterationResult) timePhase(ctx context.Context, phase Phase, run func(context.Context) error) error {
	traced, conn := withConnTrace(ctx)

	start := time.Now()
	if err := run(traced); err != nil {
		r.fail(phase, err)
		return err
	}
	r.Phases = append(r.Phases, PhaseResult{Phase: phase, Duration: time.Since(start), Connect: conn.Total()})

	return nil
}

// runUntimed runs an untimed step, recording only its error.
func (r *IterationResult) runUntimed(ctx context.Context, phase Phase, run func(context.Context) error) {
	if err := run(ctx); err != nil {
		r.fail(phase, err)
	}
}

// fail records the error of a phase, and the sandboxes it leaked.
func (r *IterationResult) fail(phase Phase, err error) {
	r.Phases = append(r.Phases, PhaseResult{Phase: phase, Err: err})

	if leaked, ok := errors.AsType[*LeakedError](err); ok {
		r.Leaked = append(r.Leaked, leaked.IDs...)
	}
}
