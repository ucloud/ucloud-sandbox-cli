package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/table"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

// listedSandbox is a display-friendly view of SandboxInfo for table rendering.
type listedSandbox struct {
	SandboxID  string    `table_field:"Sandbox ID"`
	TemplateID string    `table_field:"Template"`
	Name       string    `table_field:"Name"`
	State      string    `table_field:"State"`
	StartedAt  time.Time `table_field:"Started"`
	EndAt      time.Time `table_field:"Ends"`
	CPUCount   int       `table_field:"vCPU"`
	MemoryMB   int       `table_field:"RAM (MB)"`
}

func toListedSandbox(s sdk.SandboxInfo) listedSandbox {
	return listedSandbox{
		SandboxID:  s.SandboxID,
		TemplateID: s.TemplateID,
		Name:       s.Name,
		State:      s.State,
		StartedAt:  s.StartedAt,
		EndAt:      s.EndAt,
		CPUCount:   s.CPUCount,
		MemoryMB:   s.MemoryMB,
	}
}

func newListCmd() *cobra.Command {
	var state string
	var format string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List sandboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()
			query := &sdk.SandboxQuery{}
			if state != "" {
				query.State = []string{state}
			}

			paginator := client.ListSandboxes(ctx, query)
			var sandboxes []sdk.SandboxInfo
			for paginator.HasNext() {
				items, err := paginator.NextItems(ctx)
				if err != nil {
					return err
				}
				sandboxes = append(sandboxes, items...)
			}

			if format == "json" {
				return json.NewEncoder(os.Stdout).Encode(sandboxes)
			}

			if len(sandboxes) == 0 {
				fmt.Println("No sandboxes found.")
				return nil
			}

			rows := make([]listedSandbox, len(sandboxes))
			for i, s := range sandboxes {
				rows[i] = toListedSandbox(s)
			}

			out, err := table.Render(rows, 1, 0, int64(len(rows)))
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		},
	}

	cmd.Flags().StringVarP(&state, "state", "s", "running", "Filter by state (running, paused)")
	cmd.Flags().StringVarP(&format, "format", "f", "pretty", "Output format (pretty, json)")
	return cmd
}
