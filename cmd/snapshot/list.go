package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/table"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

// listedSnapshot is a display-friendly view of SnapshotInfo for table rendering.
type listedSnapshot struct {
	SnapshotID string `table_field:"Snapshot ID"`
	Names      string `table_field:"Names"`
}

func toListedSnapshot(s sdk.SnapshotInfo) listedSnapshot {
	names := "-"
	if len(s.Names) > 0 {
		names = strings.Join(s.Names, ", ")
	}
	return listedSnapshot{
		SnapshotID: s.SnapshotID,
		Names:      names,
	}
}

func newListCmd() *cobra.Command {
	var sandboxID string
	var format string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List snapshots",
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

			var sandboxIDPtr *string
			if sandboxID != "" {
				sandboxIDPtr = &sandboxID
			}

			paginator := client.ListSnapshots(ctx, sandboxIDPtr)
			var snapshots []sdk.SnapshotInfo
			for paginator.HasNext() {
				items, err := paginator.NextItems(ctx)
				if err != nil {
					return err
				}
				snapshots = append(snapshots, items...)
			}

			if format == "json" {
				return json.NewEncoder(os.Stdout).Encode(snapshots)
			}

			if len(snapshots) == 0 {
				fmt.Println("No snapshots found.")
				return nil
			}

			rows := make([]listedSnapshot, len(snapshots))
			for i, s := range snapshots {
				rows[i] = toListedSnapshot(s)
			}

			out, err := table.Render(rows, 1, 0, int64(len(rows)))
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		},
	}

	cmd.Flags().StringVar(&sandboxID, "sandbox-id", "", "Filter by sandbox ID")
	cmd.Flags().StringVarP(&format, "format", "f", "pretty", "Output format (pretty, json)")
	return cmd
}
