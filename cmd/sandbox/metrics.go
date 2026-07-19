package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/guptarohit/asciigraph"
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/datetime"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
	"golang.org/x/term"
)

const (
	defaultMetricsWidth = 80
	xAxisTime           = "15:04:05"
	xAxisTimeShort      = "15:04"
	xAxisHour           = "15"
	xAxisDateTime       = "01-02 15:04"
	xAxisDate           = "01-02"
	xAxisDateTimeYear   = "2006-01-02 15:04"
	xAxisDateYear       = "2006-01-02"
)

func newMetricsCmd() *cobra.Command {
	var startStr, sinceStr, endStr string
	var watch bool
	var interval int
	var raw bool

	cmd := &cobra.Command{
		Use:   "metrics <sandbox-id>",
		Short: "Show sandbox resource metrics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if startStr != "" && sinceStr != "" {
				return fmt.Errorf("--start and --since are mutually exclusive")
			}

			var start, end time.Time
			var err error
			if startStr != "" {
				if start, err = datetime.Parse(startStr); err != nil {
					return err
				}
			} else if sinceStr != "" {
				if start, err = datetime.ParseSince(sinceStr); err != nil {
					return err
				}
			}
			if endStr != "" {
				if end, err = datetime.Parse(endStr); err != nil {
					return err
				}
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()
			sbx, err := client.ConnectSandbox(ctx, args[0])
			if err != nil {
				return err
			}

			if watch {
				return watchMetrics(ctx, sbx, start, end, time.Duration(interval)*time.Second, raw)
			}
			return showMetrics(ctx, sbx, start, end, raw)
		},
	}

	cmd.Flags().StringVar(&startStr, "start", "", "Start time (e.g. '12:00', '06-23 12:00', '2025-07-23 12:00')")
	cmd.Flags().StringVar(&sinceStr, "since", "", "Start time relative to now (e.g. '1h', '30m')")
	cmd.Flags().StringVar(&endStr, "end", "", "End time (same format as --start)")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Refresh periodically")
	cmd.Flags().IntVar(&interval, "interval", 2, "Refresh interval in seconds (requires --watch)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON instead of charts")
	return cmd
}

func fetchMetrics(ctx context.Context, sbx *sdk.Sandbox, start, end time.Time) ([]sdk.SandboxMetrics, error) {
	var opts []sdk.MetricsOption
	if !start.IsZero() {
		opts = append(opts, sdk.WithMetricsStart(start))
	}
	if !end.IsZero() {
		opts = append(opts, sdk.WithMetricsEnd(end))
	}
	return sbx.GetMetrics(ctx, opts...)
}

func renderMetrics(metrics []sdk.SandboxMetrics, sandboxID string) {
	width := terminalWidth()
	if width == 0 {
		width = defaultMetricsWidth
	}
	fmt.Print(formatMetrics(metrics, sandboxID, width))
}

// terminalWidth returns stdout's usable column count. A zero result means
// stdout is not a terminal (for example, when output is redirected).
func terminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 0
	}
	return w
}

// formatMetrics renders a width-constrained metrics dashboard. Keeping this
// separate from renderMetrics makes the layout deterministic for tests and
// prevents chart libraries from expanding to the number of samples.
func formatMetrics(metrics []sdk.SandboxMetrics, sandboxID string, width int) string {
	if width <= 0 {
		width = defaultMetricsWidth
	}

	var b strings.Builder
	writeMetricLine(&b, "Sandbox metrics", width)
	idWidth := width - len("ID      ") - 2
	if idWidth < 1 {
		idWidth = 1
	}
	writeMetricLine(&b, "ID      "+truncateMetricText(sandboxID, idWidth), width)

	sampleWord := "samples"
	if len(metrics) == 1 {
		sampleWord = "sample"
	}
	writeMetricLine(&b, fmt.Sprintf("Samples %d %s", len(metrics), sampleWord), width)
	if len(metrics) == 0 {
		b.WriteByte('\n')
		writeMetricLine(&b, "No metrics data available.", width)
		return b.String()
	}

	first, last := metrics[0].Timestamp, metrics[len(metrics)-1].Timestamp
	if !first.IsZero() && !last.IsZero() {
		writeMetricLine(&b, fmt.Sprintf("Time    %s - %s", first.Format("15:04:05"), last.Format("15:04:05")), width)
	}

	b.WriteByte('\n')
	writeMetricLine(&b, "Current usage", width)
	series := buildMetricSeries(metrics)
	for _, item := range series {
		writeMetricLine(&b, item.usageLine(width), width)
	}

	b.WriteByte('\n')
	if len(metrics) < 2 {
		writeMetricLine(&b, "Waiting for more samples...", width)
		return b.String()
	}

	for _, item := range series {
		writeMetricLine(&b, "── "+item.name+" Usage (%)", width)
		for _, line := range strings.Split(formatMetricsTrendChart(item.values, first, last, width), "\n") {
			if line != "" {
				writeMetricLine(&b, line, width)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func writeMetricLine(b *strings.Builder, line string, width int) {
	b.WriteString(truncateMetricText(line, width))
	b.WriteByte('\n')
}

func truncateMetricText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

type metricSeries struct {
	name      string
	values    []float64
	current   float64
	peak      float64
	available bool
	detail    string
}

func (m metricSeries) usageLine(width int) string {
	if !m.available {
		return fmt.Sprintf("%-12sn/a  not reported", m.name)
	}
	if width >= 60 {
		return fmt.Sprintf("%-12s%6.1f%%  peak %6.1f%%  %s", m.name, m.current, m.peak, m.detail)
	}
	line := fmt.Sprintf("%-12s%.1f%%", m.name, m.current)
	if m.peak != m.current {
		line += fmt.Sprintf(" peak %.1f%%", m.peak)
	}
	if m.detail != "" {
		line += " " + m.detail
	}
	return line
}

func buildMetricSeries(metrics []sdk.SandboxMetrics) []metricSeries {
	latest := metrics[len(metrics)-1]
	cpu := make([]float64, len(metrics))
	memory := make([]float64, len(metrics))
	disk := make([]float64, len(metrics))
	for i, item := range metrics {
		cpu[i] = item.CPUUsedPct
		memory[i] = percent(item.MemUsed, item.MemTotal)
		disk[i] = percent(item.DiskUsed, item.DiskTotal)
	}
	return []metricSeries{
		{
			name:      "CPU",
			values:    cpu,
			current:   latest.CPUUsedPct,
			peak:      maxValue(cpu),
			available: true,
			detail:    fmt.Sprintf("%d vCPU", latest.CPUCount),
		},
		{
			name:      "Memory",
			values:    memory,
			current:   memory[len(memory)-1],
			peak:      maxValue(memory),
			available: latest.MemTotal > 0,
			detail:    storageDetail(latest.MemUsed, latest.MemTotal),
		},
		{
			name:      "Disk",
			values:    disk,
			current:   disk[len(disk)-1],
			peak:      maxValue(disk),
			available: latest.DiskTotal > 0,
			detail:    storageDetail(latest.DiskUsed, latest.DiskTotal),
		},
	}
}

func maxValue(values []float64) float64 {
	var max float64
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

func storageDetail(used, total int64) string {
	if total <= 0 {
		return ""
	}
	return fmt.Sprintf("%s / %s", humanize.IBytes(uint64(max(used, 0))), humanize.IBytes(uint64(total)))
}

func percent(used, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}

func formatMetricsTrendChart(values []float64, first, last time.Time, width int) string {
	// Leave room for the Y-axis labels and the plot's offset. Width is the
	// terminal width, while asciigraph's Width option is only the plot area.
	plotWidth := width - 10
	if plotWidth < 8 {
		plotWidth = 8
	}

	options := []asciigraph.Option{
		asciigraph.Width(plotWidth),
		asciigraph.Height(8),
		asciigraph.LowerBound(0),
		asciigraph.UpperBound(100),
		// Keep the original chart's percentage scale and two-decimal labels.
		asciigraph.Precision(2),
	}

	chart := asciigraph.Plot(values, options...)
	if first.IsZero() || last.IsZero() {
		return chart
	}
	return chart + "\n" + formatMetricsXAxis(chart, plotWidth, first, last)
}

func formatMetricsXAxis(chart string, plotWidth int, first, last time.Time) string {
	lines := strings.Split(chart, "\n")
	axisColumn := -1
	for _, line := range lines {
		if column := strings.IndexRune(line, '┤'); column >= 0 {
			axisColumn = column + 1
			break
		}
	}
	if axisColumn < 1 {
		return ""
	}

	timeFormat, tickCount := metricsXAxisLayout(plotWidth, first, last)

	lineWidth := axisColumn + plotWidth
	axis := []rune(strings.Repeat(" ", lineWidth))
	axis[axisColumn-1] = '└'
	for i := 0; i < plotWidth; i++ {
		axis[axisColumn+i] = '─'
	}

	labelLine := []rune(strings.Repeat(" ", lineWidth))
	labelWidth := len([]rune(first.Format(timeFormat)))
	labelRange := plotWidth - labelWidth
	for i := 0; i < tickCount; i++ {
		fraction := float64(i) / float64(tickCount-1)
		position := int(float64(plotWidth-1) * fraction)
		axis[axisColumn+position] = '┬'

		stamp := first.Add(time.Duration(float64(last.Sub(first)) * fraction))
		label := []rune(stamp.Format(timeFormat))
		start := axisColumn + labelRange*i/(tickCount-1)
		copy(labelLine[start:], label)
	}

	return strings.TrimRight(string(axis), " ") + "\n" + strings.TrimRight(string(labelLine), " ")
}

func metricsXAxisLayout(plotWidth int, first, last time.Time) (string, int) {
	formats := []string{xAxisTime, xAxisTimeShort, xAxisHour}
	if first.Year() != last.Year() {
		formats = []string{xAxisDateTimeYear, xAxisDateYear, xAxisDate, xAxisHour}
	} else if first.YearDay() != last.YearDay() {
		formats = []string{xAxisDateTime, xAxisDate, xAxisTimeShort, xAxisHour}
	}

	for _, format := range formats {
		maxTicks := (plotWidth + 1) / (len(format) + 1)
		switch {
		case maxTicks >= 10:
			return format, 10
		case maxTicks >= 5:
			return format, 5
		case maxTicks >= 2:
			return format, 2
		}
	}
	return formats[len(formats)-1], 2
}

func showMetrics(ctx context.Context, sbx *sdk.Sandbox, start, end time.Time, raw bool) error {
	metrics, err := fetchMetrics(ctx, sbx, start, end)
	if err != nil {
		return err
	}
	if raw {
		return json.NewEncoder(os.Stdout).Encode(metrics)
	}
	renderMetrics(metrics, sbx.ID)
	return nil
}

func watchMetrics(ctx context.Context, sbx *sdk.Sandbox, start, end time.Time, interval time.Duration, raw bool) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	tick := time.NewTicker(interval)
	defer tick.Stop()

	render := func() error {
		metrics, err := fetchMetrics(ctx, sbx, start, end)
		if err != nil {
			return err
		}
		if raw {
			return json.NewEncoder(os.Stdout).Encode(metrics)
		}
		fmt.Print("\033[H\033[2J") // clear screen
		fmt.Printf("Refreshing every %v  (Ctrl+C to stop)\n\n", interval)
		renderMetrics(metrics, sbx.ID)
		return nil
	}

	if err := render(); err != nil {
		return err
	}
	for {
		select {
		case <-tick.C:
			if err := render(); err != nil {
				return err
			}
		case <-sigCh:
			return nil
		}
	}
}
