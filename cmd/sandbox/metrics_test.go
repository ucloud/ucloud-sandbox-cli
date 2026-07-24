package sandbox

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

func TestFormatMetricsDashboard(t *testing.T) {
	start := time.Date(2026, time.July, 17, 14, 30, 0, 0, time.Local)
	metrics := []sdk.SandboxMetrics{
		{
			Timestamp:  start,
			CPUCount:   2,
			CPUUsedPct: 10,
			MemUsed:    256 << 20,
			MemTotal:   1 << 30,
			DiskUsed:   2 << 30,
			DiskTotal:  10 << 30,
		},
		{
			Timestamp:  start.Add(time.Minute),
			CPUCount:   2,
			CPUUsedPct: 25,
			MemUsed:    768 << 20,
			MemTotal:   1 << 30,
			DiskUsed:   3 << 30,
			DiskTotal:  10 << 30,
		},
		{
			Timestamp:  start.Add(2 * time.Minute),
			CPUCount:   2,
			CPUUsedPct: 20,
			MemUsed:    512 << 20,
			MemTotal:   1 << 30,
			DiskUsed:   4 << 30,
			DiskTotal:  10 << 30,
		},
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
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\033[") {
		t.Fatalf("color-disabled output contains ANSI escapes:\n%s", out)
	}
	assertMetricsLineWidth(t, out, 80)
}

func TestFormatMetricsNarrowAndUnavailable(t *testing.T) {
	metrics := []sdk.SandboxMetrics{{
		Timestamp:  time.Date(2026, time.July, 17, 14, 30, 0, 0, time.Local),
		CPUCount:   1,
		CPUUsedPct: 8.5,
	}}

	out := formatMetrics(metrics, "sandbox-with-a-very-long-identifier", 32)

	for _, want := range []string{
		"ID      sandbox-with-a-very...",
		"Memory      n/a  not reported",
		"Disk        n/a  not reported",
		"Waiting for more samples...",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "[█") || strings.Contains(out, "── CPU Usage (%)") {
		t.Fatalf("narrow single-sample output should use the compact layout:\n%s", out)
	}
	assertMetricsLineWidth(t, out, 32)
}

func TestFormatMetricsWideXAxisKeepsEndpointLabels(t *testing.T) {
	start := time.Date(2026, time.July, 17, 17, 20, 0, 0, time.Local)
	end := time.Date(2026, time.July, 17, 19, 45, 0, 0, time.Local)
	metrics := []sdk.SandboxMetrics{
		{Timestamp: start, CPUUsedPct: 10, MemTotal: 1, DiskTotal: 1},
		{Timestamp: end, CPUUsedPct: 20, MemTotal: 1, DiskTotal: 1},
	}

	out := formatMetrics(metrics, "sandbox-123", 100)
	for _, label := range []string{"17:20:00", "19:45:00"} {
		if count := strings.Count(out, label); count != 4 {
			t.Errorf("time label %q appears %d times, want once in the summary and each chart:\n%s", label, count, out)
		}
	}
	assertMetricsLineWidth(t, out, 100)
}

func TestMetricsXAxisLayoutIncludesDateWhenNeeded(t *testing.T) {
	dayStart := time.Date(2026, time.July, 17, 23, 30, 0, 0, time.Local)
	dayEnd := dayStart.Add(2 * time.Hour)
	format, ticks := metricsXAxisLayout(100, dayStart, dayEnd)
	if format != "01-02 15:04" || ticks != 5 {
		t.Fatalf("cross-day layout = (%q, %d), want (%q, %d)", format, ticks, "01-02 15:04", 5)
	}

	yearEnd := time.Date(2027, time.January, 1, 0, 30, 0, 0, time.Local)
	format, ticks = metricsXAxisLayout(100, dayEnd, yearEnd)
	if format != "2006-01-02 15:04" || ticks != 5 {
		t.Fatalf("cross-year layout = (%q, %d), want (%q, %d)", format, ticks, "2006-01-02 15:04", 5)
	}
}

func TestMetricDisplayRangeUsesLocalTimezone(t *testing.T) {
	sourceLocation := time.FixedZone("source", 2*60*60)
	firstInput := time.Date(2026, time.July, 17, 23, 30, 0, 0, sourceLocation)
	lastInput := firstInput.Add(2 * time.Hour)
	first, last := metricDisplayRange([]sdk.SandboxMetrics{
		{Timestamp: firstInput},
		{Timestamp: lastInput},
	})

	wantFirst := firstInput.In(time.Local)
	wantLast := lastInput.In(time.Local)
	if !first.Equal(wantFirst) || first.Location() != time.Local {
		t.Fatalf("first display time = %v (%v), want %v (%v)", first, first.Location(), wantFirst, time.Local)
	}
	if !last.Equal(wantLast) || last.Location() != time.Local {
		t.Fatalf("last display time = %v (%v), want %v (%v)", last, last.Location(), wantLast, time.Local)
	}
}

func assertMetricsLineWidth(t *testing.T, output string, maxWidth int) {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		if width := utf8.RuneCountInString(line); width > maxWidth {
			t.Errorf("line is %d columns wide, want at most %d: %q", width, maxWidth, line)
		}
	}
}
