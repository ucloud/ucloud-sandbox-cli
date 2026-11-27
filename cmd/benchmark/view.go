package benchmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ucloud/ucloud-sandbox-cli/internal/benchmark"
	"github.com/ucloud/ucloud-sandbox-cli/internal/table"
)

// runView is the output of a benchmark run, shaped for JSON with durations in
// milliseconds.
type runView struct {
	RunID             string `json:"run_id"`
	Template          string `json:"template"`
	Concurrency       int    `json:"concurrency"`
	SandboxTimeoutSec int32  `json:"sandbox_timeout_sec"`

	Requested   int  `json:"requested"`
	Started     int  `json:"started"`
	Succeeded   int  `json:"succeeded"`
	Failed      int  `json:"failed"`
	Interrupted bool `json:"interrupted"`

	ElapsedMs float64 `json:"elapsed_ms"`
	Rate      float64 `json:"rate_per_sec"`

	Phases []phaseView `json:"phases"`
	Errors []errorView `json:"errors"`
	Leaked []string    `json:"leaked"`

	// LeakedSnapshots names the snapshots the run took but could not delete.
	// The names, not the IDs, are what `snapshot delete` deletes them by.
	LeakedSnapshots []string `json:"leaked_snapshots"`

	// LeftRunning counts the sandboxes the run left running on purpose, to be
	// reclaimed by their timeout.
	LeftRunning int `json:"left_running"`

	// Iterations is the raw sample of every iteration that ran. It is only
	// written as JSON.
	Iterations []iterationView `json:"iterations"`

	// breakdown shows every phase's full latency in the pretty table, with its
	// connect and service rows under it, rather than its service time alone.
	breakdown bool

	// copyStartsFailed counts the restores and forks that failed after their
	// request was sent, and so may have started a sandbox all the same.
	copyStartsFailed int
}

// phaseView is the statistics of one phase. Its latencies cover the whole
// phase; Connect and Service split every sample into the time spent obtaining
// HTTP connections and the rest.
type phaseView struct {
	Phase     string `json:"phase"`
	Succeeded int    `json:"succeeded"`
	Failed    int    `json:"failed"`
	latencyView
	Connect latencyView `json:"connect"`
	Service latencyView `json:"service"`
}

type latencyView struct {
	AvgMs float64 `json:"avg_ms"`
	MinMs float64 `json:"min_ms"`
	P50Ms float64 `json:"p50_ms"`
	P95Ms float64 `json:"p95_ms"`
	P99Ms float64 `json:"p99_ms"`
	MaxMs float64 `json:"max_ms"`
}

func newLatencyView(s benchmark.Stats) latencyView {
	return latencyView{
		AvgMs: ms(s.Avg),
		MinMs: ms(s.Min),
		P50Ms: ms(s.P50),
		P95Ms: ms(s.P95),
		P99Ms: ms(s.P99),
		MaxMs: ms(s.Max),
	}
}

// iterationView is the outcome of one iteration.
type iterationView struct {
	Seq           int     `json:"seq"`
	SandboxID     string  `json:"sandbox_id"`
	StartOffsetMs float64 `json:"start_offset_ms"`
	Succeeded     bool    `json:"succeeded"`

	Phases []iterationPhaseView `json:"phases"`
	Leaked []string             `json:"leaked"`
}

// iterationPhaseView is one phase of an iteration: its latencies when it
// succeeded, its error when it failed. The latencies are pointers so a zero
// connect time is still written.
type iterationPhaseView struct {
	Phase      string   `json:"phase"`
	DurationMs *float64 `json:"duration_ms,omitempty"`
	ConnectMs  *float64 `json:"connect_ms,omitempty"`
	Error      string   `json:"error,omitempty"`
}

func newIterationView(r benchmark.IterationResult) iterationView {
	v := iterationView{
		Seq:           r.Seq,
		SandboxID:     r.SandboxID,
		StartOffsetMs: ms(r.Start),
		Succeeded:     r.Succeeded(),
		Phases:        make([]iterationPhaseView, 0, len(r.Phases)),
		Leaked:        append([]string{}, r.Leaked...),
	}

	for _, p := range r.Phases {
		if p.Err != nil {
			// Collapse whitespace so the message fits on one line. Unlike the
			// grouped errors, the sandbox ID is kept: it is this iteration's.
			v.Phases = append(v.Phases, iterationPhaseView{
				Phase: string(p.Phase),
				Error: strings.Join(strings.Fields(p.Err.Error()), " "),
			})
			continue
		}
		duration, connect := ms(p.Duration), ms(p.Connect)
		v.Phases = append(v.Phases, iterationPhaseView{
			Phase:      string(p.Phase),
			DurationMs: &duration,
			ConnectMs:  &connect,
		})
	}

	return v
}

type errorView struct {
	Phase   string `json:"phase"`
	Count   int    `json:"count"`
	Message string `json:"message"`
}

func newRunView(runID, template string, concurrency int, timeoutSec int32, r benchmark.Report) runView {
	v := runView{
		RunID:             runID,
		Template:          template,
		Concurrency:       concurrency,
		SandboxTimeoutSec: timeoutSec,
		Requested:         r.Requested,
		Started:           r.Started,
		Succeeded:         r.Succeeded,
		Failed:            r.Failed,
		Interrupted:       r.Interrupted,
		ElapsedMs:         ms(r.Elapsed),
		Rate:              r.Rate(),
		Phases:            make([]phaseView, 0, len(r.Phases)),
		Errors:            make([]errorView, 0, len(r.Errors)),
		Leaked:            append([]string{}, r.Leaked...),
		LeakedSnapshots:   []string{},
		Iterations:        make([]iterationView, 0, len(r.Iterations)),
	}

	for _, p := range r.Phases {
		v.Phases = append(v.Phases, phaseView{
			Phase:       string(p.Phase),
			Succeeded:   p.Stats.Count,
			Failed:      p.Failed,
			latencyView: newLatencyView(p.Stats),
			Connect:     newLatencyView(p.Connect),
			Service:     newLatencyView(p.Service),
		})
	}

	for _, e := range r.Errors {
		v.Errors = append(v.Errors, errorView{Phase: string(e.Phase), Count: e.Count, Message: e.Message})
	}

	for _, it := range r.Iterations {
		v.Iterations = append(v.Iterations, newIterationView(it))
	}

	v.copyStartsFailed = failedRequests(r.Iterations, phaseRestore, phaseFork)

	return v
}

// ms converts d to milliseconds, rounded to the microsecond.
func ms(d time.Duration) float64 {
	return float64(d.Round(time.Microsecond)) / float64(time.Millisecond)
}

// phaseRow is one row of the latency table. The counts are strings so the
// breakdown rows under a phase can leave them blank.
type phaseRow struct {
	Phase     string `table_field:"Phase"`
	Succeeded string `table_field:"Succeeded"`
	Failed    string `table_field:"Failed"`
	Avg       string `table_field:"Avg"`
	Min       string `table_field:"Min"`
	P50       string `table_field:"P50"`
	P95       string `table_field:"P95"`
	P99       string `table_field:"P99"`
	Max       string `table_field:"Max"`
}

// withLatency fills in the latency columns of the row.
func (r phaseRow) withLatency(l latencyView) phaseRow {
	r.Avg, r.Min = formatMs(l.AvgMs), formatMs(l.MinMs)
	r.P50, r.P95, r.P99, r.Max = formatMs(l.P50Ms), formatMs(l.P95Ms), formatMs(l.P99Ms), formatMs(l.MaxMs)
	return r
}

// errorRow is one row of the error table.
type errorRow struct {
	Phase   string `table_field:"Phase"`
	Count   int    `table_field:"Count"`
	Message string `table_field:"Message"`
}

// checkReportFile checks that path can be written, creating it if it does not
// exist. It runs before a benchmark starts, so a wrong path fails in the first
// second instead of after the whole run.
func checkReportFile(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("report file: %w", err)
	}
	return f.Close()
}

// writeJSONFile writes the view to path as the same JSON document --format
// json prints, so a run can show its table and keep its samples at once.
func (v runView) writeJSONFile(path string) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

// writePretty writes the view as a summary followed by ASCII tables.
func (v runView) writePretty(w io.Writer) error {
	status := "completed"
	if v.Interrupted {
		status = "interrupted"
	}

	fmt.Fprintf(w, "Benchmark %s %s: template %s, concurrency %d\n", v.RunID, status, v.Template, v.Concurrency)
	fmt.Fprintf(w, "Iterations: %d/%d started, %d succeeded, %d failed\n", v.Started, v.Requested, v.Succeeded, v.Failed)
	fmt.Fprintf(w, "Elapsed: %s, rate: %s sandbox/s\n\n",
		formatMs(v.ElapsedMs), strconv.FormatFloat(v.Rate, 'f', 2, 64))

	// The JSON output reports the full latency under the same names, so say
	// which one the table shows.
	if v.breakdown {
		fmt.Fprintln(w, "Latency per phase, split into connection setup (connect) and service time (service):")
	} else {
		fmt.Fprintln(w, "Service time per phase, excluding connection setup (DNS, dial, TLS); --breakdown shows the full latency:")
	}

	// By default a phase shows its service time: its latency less the time
	// spent obtaining HTTP connections (DNS, dial, TLS), which depends on the
	// client's network more than on the service. The breakdown shows the full
	// latency, followed by both parts.
	rows := make([]phaseRow, 0, 3*len(v.Phases))
	for _, p := range v.Phases {
		row := phaseRow{Phase: p.Phase, Succeeded: strconv.Itoa(p.Succeeded), Failed: strconv.Itoa(p.Failed)}
		if p.Succeeded == 0 {
			rows = append(rows, row)
			continue
		}
		if !v.breakdown {
			rows = append(rows, row.withLatency(p.Service))
			continue
		}
		rows = append(rows,
			row.withLatency(p.latencyView),
			phaseRow{Phase: "  connect"}.withLatency(p.Connect),
			phaseRow{Phase: "  service"}.withLatency(p.Service),
		)
	}

	out, err := table.RenderPlain(rows)
	if err != nil {
		return err
	}
	fmt.Fprint(w, out)

	if len(v.Errors) > 0 {
		errs := make([]errorRow, 0, len(v.Errors))
		for _, e := range v.Errors {
			errs = append(errs, errorRow(e))
		}

		out, err := table.RenderPlain(errs)
		if err != nil {
			return err
		}
		fmt.Fprint(w, "\nErrors:\n"+out)
	}

	v.writeCleanup(w)
	v.writeSnapshotCleanup(w)

	return nil
}

// writeCleanup tells the user about every sandbox the run may have left
// behind, followed by the one command that kills them all.
func (v runView) writeCleanup(w io.Writer) {
	hint := ""

	if len(v.Leaked) > 0 {
		fmt.Fprintf(w, "\n%d sandbox(es) could not be killed and live until their timeout:\n", len(v.Leaked))
		for _, id := range v.Leaked {
			fmt.Fprintf(w, "  %s\n", id)
		}
		// Killing by ID also reaches a sandbox without the run's tag, which
		// a fork may be.
		fmt.Fprintf(w, "Kill them with: ucloud-sandbox-cli sandbox kill %s\n", strings.Join(v.Leaked, " "))
		hint = "Clean up the rest of the run with"
	}

	// A create that failed on the client, a timeout say, may still have
	// started its sandbox on the service, where nothing here can see it.
	if failed := v.phaseFailed(benchmark.PhaseCreate); failed > 0 {
		fmt.Fprintf(w, "\n%d create(s) failed; the service may still have started some of those sandboxes.\n", failed)
		hint = "Clean them up with"
	}

	// Restoring and forking start sandboxes too, the same way.
	if failed := v.copyStartsFailed; failed > 0 {
		fmt.Fprintf(w, "\n%d restore(s) or fork(s) failed; the service may still have started some of those sandboxes.\n", failed)
		hint = "Clean them up with"
	}

	if v.LeftRunning > 0 {
		fmt.Fprintf(w, "\n%d sandbox(es) left running, reclaimed after their %s timeout.\n",
			v.LeftRunning, time.Duration(v.SandboxTimeoutSec)*time.Second)
		if hint == "" {
			hint = "Kill them sooner with"
		}
	}

	if hint != "" {
		fmt.Fprintf(w, "%s: %s\n", hint, cleanupCommand(v.RunID))
	}
}

// phaseFailed returns the number of iterations in which phase failed.
func (v runView) phaseFailed(phase benchmark.Phase) int {
	for _, p := range v.Phases {
		if p.Phase == string(phase) {
			return p.Failed
		}
	}
	return 0
}

// failedRequests counts the failures of phases that got as far as sending
// their request. A step left out because one before it failed sent nothing,
// so it cannot have started anything on the service.
func failedRequests(iterations []benchmark.IterationResult, phases ...benchmark.Phase) int {
	n := 0
	for _, it := range iterations {
		for _, p := range it.Phases {
			if p.Err != nil && slices.Contains(phases, p.Phase) && !errors.Is(p.Err, errNotAttempted) {
				n++
			}
		}
	}
	return n
}

// writeSnapshotCleanup tells the user about the snapshots the run may have
// left behind. Unlike sandboxes, they never expire.
func (v runView) writeSnapshotCleanup(w io.Writer) {
	if len(v.LeakedSnapshots) > 0 {
		fmt.Fprintf(w, "\n%d snapshot(s) could not be deleted and are kept until deleted:\n", len(v.LeakedSnapshots))
		for _, id := range v.LeakedSnapshots {
			fmt.Fprintf(w, "  %s\n", id)
		}
		fmt.Fprintf(w, "Delete them with: ucloud-sandbox-cli snapshot delete %s\n", strings.Join(v.LeakedSnapshots, " "))
	}

	// Like a create, a snapshot that failed on the client may still have been
	// taken on the service.
	if failed := v.phaseFailed(phaseSnapshot); failed > 0 {
		fmt.Fprintf(w, "\n%d snapshot(s) failed; the service may still have taken some of them.\n", failed)
		fmt.Fprintf(w, "Look for snapshots whose names contain %s in: ucloud-sandbox-cli snapshot list\n", snapshotNamePrefix(v.RunID))
	}
}

// formatMs renders a millisecond value as a duration rounded to the
// millisecond, e.g. "523ms" or "1.234s".
func formatMs(v float64) string {
	return time.Duration(v * float64(time.Millisecond)).Round(time.Millisecond).String()
}
