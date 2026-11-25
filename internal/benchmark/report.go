package benchmark

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// Report aggregates the results of a benchmark run.
type Report struct {
	// Requested is the number of iterations asked for; Started is the number
	// that actually ran, which is lower when the run was interrupted.
	Requested int
	Started   int

	Succeeded int
	Failed    int

	Interrupted bool
	Elapsed     time.Duration

	// Phases holds the statistics of every timed phase, in execution order and
	// followed by PhaseTotal when the run timed more than one.
	Phases []PhaseReport

	// Errors groups the failures by phase and message, most frequent first.
	Errors []ErrorCount

	// Leaked lists the sandboxes that were created but could not be killed.
	Leaked []string
}

// PhaseReport is the latency statistics of one phase. Stats.Count is the
// number of times the phase succeeded.
//
// Every successful sample is split in two: Connect is the time spent
// obtaining HTTP connections (DNS, dial, TLS), and Service is the rest, the
// time the service took to respond. Each is summarized on its own, so their
// percentiles do not add up to those of Stats.
type PhaseReport struct {
	Phase   Phase
	Failed  int
	Stats   Stats
	Connect Stats
	Service Stats
}

// ErrorCount is the number of times one error occurred in one phase.
type ErrorCount struct {
	Phase   Phase
	Message string
	Count   int
}

// Throughput returns the number of successful iterations per second.
func (r Report) Throughput() float64 {
	if r.Elapsed <= 0 {
		return 0
	}
	return float64(r.Succeeded) / r.Elapsed.Seconds()
}

func buildReport(
	results []IterationResult,
	requested int,
	phases []Phase,
	interrupted bool,
	elapsed time.Duration,
) Report {

	report := Report{
		Requested:   requested,
		Started:     len(results),
		Interrupted: interrupted,
		Elapsed:     elapsed,
	}

	samples := make(map[Phase][]time.Duration, len(phases))
	connects := make(map[Phase][]time.Duration, len(phases))
	services := make(map[Phase][]time.Duration, len(phases))
	failed := make(map[Phase]int, len(phases))

	type errorKey struct {
		phase   Phase
		message string
	}
	errorCounts := make(map[errorKey]int)
	var errorOrder []errorKey

	for _, r := range results {
		if r.Succeeded() {
			report.Succeeded++
		} else {
			report.Failed++
			failed[PhaseTotal]++
		}

		for phase, d := range r.Durations {
			connect := r.Connects[phase]
			samples[phase] = append(samples[phase], d)
			connects[phase] = append(connects[phase], connect)
			services[phase] = append(services[phase], d-connect)
		}

		for _, e := range r.Errors {
			failed[e.Phase]++

			key := errorKey{phase: e.Phase, message: normalizeMessage(e.Err.Error(), r.SandboxID)}
			if errorCounts[key] == 0 {
				errorOrder = append(errorOrder, key)
			}
			errorCounts[key]++
		}

		if r.Leaked {
			report.Leaked = append(report.Leaked, r.SandboxID)
		}
	}

	for _, phase := range phases {
		report.Phases = append(report.Phases, PhaseReport{
			Phase:   phase,
			Failed:  failed[phase],
			Stats:   Summarize(samples[phase]),
			Connect: Summarize(connects[phase]),
			Service: Summarize(services[phase]),
		})
	}

	for _, key := range errorOrder {
		report.Errors = append(report.Errors, ErrorCount{
			Phase:   key.phase,
			Message: key.message,
			Count:   errorCounts[key],
		})
	}
	// Stable, so errors seen equally often keep the order they first appeared.
	slices.SortStableFunc(report.Errors, func(a, b ErrorCount) int {
		return cmp.Compare(b.Count, a.Count)
	})

	slices.Sort(report.Leaked)

	return report
}

// sandboxPlaceholder stands in for the sandbox ID in a normalized message.
const sandboxPlaceholder = "<sandbox>"

// normalizeMessage prepares an error message for grouping and display. The
// iteration's sandbox ID, which turns up in hosts such as
// "49983-<id>.<domain>", is replaced by a placeholder so the same failure in
// different sandboxes groups together, and runs of whitespace, newlines
// included, collapse to one space so the message fits on one table row.
func normalizeMessage(msg, sandboxID string) string {
	if sandboxID != "" {
		msg = strings.ReplaceAll(msg, sandboxID, sandboxPlaceholder)
	}
	return strings.Join(strings.Fields(msg), " ")
}
