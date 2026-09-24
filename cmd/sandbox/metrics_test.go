package sandbox

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// parseMetricsFlags registers the metrics flags the way the command does and
// returns the operation they filled in.
func parseMetricsFlags(t *testing.T, args ...string) *metricsOperation {
	t.Helper()

	op := &metricsOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))

	return op
}

func TestMetricsFlags(t *testing.T) {
	op := parseMetricsFlags(t,
		"--start", "1758693600",
		"--end", "1758697200",
		"-w", "--interval", "5", "--raw",
		"sbx-1",
	)

	require.NotNil(t, op.params.Start)
	assert.Equal(t, int64(1758693600), *op.params.Start)
	require.NotNil(t, op.params.End)
	assert.Equal(t, int64(1758697200), *op.params.End)

	assert.True(t, op.watch)
	assert.Equal(t, 5, op.interval)
	assert.True(t, op.raw)
}

func TestMetricsIntervalDefaultsAndBoundsStayUnset(t *testing.T) {
	op := parseMetricsFlags(t, "sbx-1")

	assert.Nil(t, op.params.Start, "an absent --start leaves the window open")
	assert.Nil(t, op.params.End)
	assert.False(t, op.watch)
	assert.Equal(t, 2, op.interval)
}

// sample builds one metric sample taken at stamp.
func sample(stamp time.Time, cpuPct float32, memUsed, memTotal, diskUsed, diskTotal int64) api.SandboxMetric {
	return api.SandboxMetric{
		TimestampUnix: stamp.Unix(),
		CpuCount:      2,
		CpuUsedPct:    cpuPct,
		MemUsed:       memUsed,
		MemTotal:      memTotal,
		DiskUsed:      diskUsed,
		DiskTotal:     diskTotal,
	}
}

func TestMetricTimePrefersTheUnixTimestamp(t *testing.T) {
	stamp := time.Date(2026, time.July, 17, 14, 30, 0, 0, time.Local)

	// The deprecated Timestamp is only the fallback.
	assert.True(t, metricTime(api.SandboxMetric{TimestampUnix: stamp.Unix()}).Equal(stamp))
	assert.True(t, metricTime(api.SandboxMetric{Timestamp: stamp}).Equal(stamp))
	assert.True(t, metricTime(api.SandboxMetric{
		TimestampUnix: stamp.Unix(),
		Timestamp:     stamp.Add(time.Hour),
	}).Equal(stamp))
}

func TestFormatMetricsDashboard(t *testing.T) {
	start := time.Date(2026, time.July, 17, 14, 30, 0, 0, time.Local)
	metrics := []api.SandboxMetric{
		sample(start, 10, 256<<20, 1<<30, 2<<30, 10<<30),
		sample(start.Add(time.Minute), 25, 768<<20, 1<<30, 3<<30, 10<<30),
		sample(start.Add(2*time.Minute), 20, 512<<20, 1<<30, 4<<30, 10<<30),
	}

	out := formatMetrics(metrics, "sandbox-123", 80)

	for _, want := range []string{
		"Sandbox metrics",
		"ID      sandbox-123",
		"3 samples",
		"Current usage",
		"CPU",
		"  20.0%  peak   25.0%  2 vCPU",
		"Memory",
		"  50.0%  peak   75.0%  512 MiB / 1.0 GiB",
		"Disk",
		"  40.0%  peak   40.0%  4.0 GiB / 10 GiB",
		"── CPU Usage (%)",
		"── Memory Usage (%)",
		"── Disk Usage (%)",
	} {
		assert.Contains(t, out, want)
	}

	assert.NotContains(t, out, "\033[", "color-disabled output must carry no ANSI escapes")
	assertMetricsLineWidth(t, out, 80)
}

func TestFormatMetricsWithoutSamples(t *testing.T) {
	out := formatMetrics(nil, "sandbox-123", 80)

	assert.Contains(t, out, "Samples 0 samples")
	assert.Contains(t, out, "No metrics data available.")
}

func TestFormatMetricsNarrowAndUnavailable(t *testing.T) {
	metrics := []api.SandboxMetric{{
		TimestampUnix: time.Date(2026, time.July, 17, 14, 30, 0, 0, time.Local).Unix(),
		CpuCount:      1,
		CpuUsedPct:    8.5,
	}}

	out := formatMetrics(metrics, "sandbox-with-a-very-long-identifier", 32)

	for _, want := range []string{
		"ID      sandbox-with-a-very...",
		"Memory      n/a  not reported",
		"Disk        n/a  not reported",
		"Waiting for more samples...",
	} {
		assert.Contains(t, out, want)
	}

	assert.NotContains(t, out, "[█", "a narrow single-sample dashboard uses the compact layout")
	assert.NotContains(t, out, "── CPU Usage (%)")
	assertMetricsLineWidth(t, out, 32)
}

func TestFormatMetricsWideXAxisKeepsEndpointLabels(t *testing.T) {
	start := time.Date(2026, time.July, 17, 17, 20, 0, 0, time.Local)
	end := time.Date(2026, time.July, 17, 19, 45, 0, 0, time.Local)
	metrics := []api.SandboxMetric{
		sample(start, 10, 0, 1, 0, 1),
		sample(end, 20, 0, 1, 0, 1),
	}

	out := formatMetrics(metrics, "sandbox-123", 100)

	// Once in the summary line, then once per chart.
	for _, label := range []string{"17:20:00", "19:45:00"} {
		assert.Equal(t, 4, strings.Count(out, label), "time label %q", label)
	}

	assertMetricsLineWidth(t, out, 100)
}

func TestMetricsXAxisLayoutIncludesDateWhenNeeded(t *testing.T) {
	dayStart := time.Date(2026, time.July, 17, 23, 30, 0, 0, time.Local)
	dayEnd := dayStart.Add(2 * time.Hour)

	format, ticks := metricsXAxisLayout(100, dayStart, dayEnd)
	assert.Equal(t, xAxisDateTime, format)
	assert.Equal(t, 5, ticks)

	yearEnd := time.Date(2027, time.January, 1, 0, 30, 0, 0, time.Local)

	format, ticks = metricsXAxisLayout(100, dayEnd, yearEnd)
	assert.Equal(t, xAxisDateTimeYear, format)
	assert.Equal(t, 5, ticks)
}

func TestMetricDisplayRangeUsesLocalTimezone(t *testing.T) {
	sourceLocation := time.FixedZone("source", 2*60*60)
	firstInput := time.Date(2026, time.July, 17, 23, 30, 0, 0, sourceLocation)
	lastInput := firstInput.Add(2 * time.Hour)

	first, last := metricDisplayRange([]api.SandboxMetric{
		{Timestamp: firstInput},
		{Timestamp: lastInput},
	})

	assert.True(t, first.Equal(firstInput))
	assert.Equal(t, time.Local, first.Location())
	assert.True(t, last.Equal(lastInput))
	assert.Equal(t, time.Local, last.Location())
}

func assertMetricsLineWidth(t *testing.T, output string, maxWidth int) {
	t.Helper()

	for _, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		assert.LessOrEqual(t, utf8.RuneCountInString(line), maxWidth, "line %q", line)
	}
}
