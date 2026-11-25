package benchmark

import (
	"math"
	"slices"
	"time"
)

// Stats summarizes a set of latency samples.
type Stats struct {
	Count int
	Avg   time.Duration
	Min   time.Duration
	P50   time.Duration
	P95   time.Duration
	P99   time.Duration
	Max   time.Duration
}

// Summarize computes the latency statistics of samples. An empty input yields
// zero Stats. The input slice is left untouched.
func Summarize(samples []time.Duration) Stats {
	if len(samples) == 0 {
		return Stats{}
	}

	sorted := slices.Clone(samples)
	slices.Sort(sorted)

	var sum time.Duration
	for _, s := range sorted {
		sum += s
	}

	return Stats{
		Count: len(sorted),
		Avg:   sum / time.Duration(len(sorted)),
		Min:   sorted[0],
		P50:   percentile(sorted, 50),
		P95:   percentile(sorted, 95),
		P99:   percentile(sorted, 99),
		Max:   sorted[len(sorted)-1],
	}
}

// percentile returns the p-th percentile of sorted using the nearest-rank
// method, so the result is always one of the observed samples.
func percentile(sorted []time.Duration, p float64) time.Duration {
	rank := int(math.Ceil(p / 100 * float64(len(sorted))))
	rank = min(max(rank, 1), len(sorted))
	return sorted[rank-1]
}
