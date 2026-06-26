package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guptarohit/asciigraph"
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/datetime"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
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
	if len(metrics) == 0 {
		fmt.Println("No metrics data available.")
		return
	}

	cpu := make([]float64, len(metrics))
	mem := make([]float64, len(metrics))
	disk := make([]float64, len(metrics))

	for i, m := range metrics {
		cpu[i] = m.CPUUsedPct
		if m.MemTotal > 0 {
			mem[i] = float64(m.MemUsed) / float64(m.MemTotal) * 100
		}
		if m.DiskTotal > 0 {
			disk[i] = float64(m.DiskUsed) / float64(m.DiskTotal) * 100
		}
	}

	tStart := float64(metrics[0].Timestamp.Unix())
	tEnd := float64(metrics[len(metrics)-1].Timestamp.Unix())
	timeFormatter := func(v float64) string {
		return time.Unix(int64(v), 0).Format("15:04:05")
	}

	chartOpts := []asciigraph.Option{
		asciigraph.Height(8),
		asciigraph.LowerBound(0),
		asciigraph.UpperBound(100),
		asciigraph.XAxisRange(tStart, tEnd),
		asciigraph.XAxisValueFormatter(timeFormatter),
	}

	fmt.Printf("Sandbox: %s  (%d samples)\n\n", sandboxID, len(metrics))
	fmt.Println("── CPU Usage (%)")
	fmt.Println(asciigraph.Plot(cpu, chartOpts...))
	fmt.Println()
	fmt.Println("── Memory Usage (%)")
	fmt.Println(asciigraph.Plot(mem, chartOpts...))
	fmt.Println()
	fmt.Println("── Disk Usage (%)")
	fmt.Println(asciigraph.Plot(disk, chartOpts...))
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
