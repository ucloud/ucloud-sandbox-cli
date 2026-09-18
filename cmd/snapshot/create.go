package snapshot

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

func newCreateCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "create <sandbox-id>",
		Aliases: []string{"cr"},
		Short:   "Create a snapshot from a sandbox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Connect to the sandbox first
			sbx, err := client.ConnectSandbox(ctx, sandboxID)
			if err != nil {
				return fmt.Errorf("failed to connect to sandbox %s: %w", sandboxID, err)
			}

			var opts []sdk.SnapshotOption
			if name != "" {
				opts = append(opts, sdk.WithSnapshotName(name))
			}

			// Create snapshot from the sandbox
			snapshot, err := sbx.CreateSnapshot(ctx, opts...)
			if err != nil {
				return fmt.Errorf("failed to create snapshot: %w", err)
			}

			if name != "" {
				fmt.Printf("Snapshot created: %s (%s)\n", snapshot.SnapshotID, name)
			} else {
				fmt.Printf("Snapshot created: %s\n", snapshot.SnapshotID)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Snapshot name")

	return cmd
}
