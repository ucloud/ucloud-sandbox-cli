package sandbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

func newCloneCmd() *cobra.Command {
	var timeout int
	var detach bool

	cmd := &cobra.Command{
		Use:     "clone <sandbox-id>",
		Aliases: []string{"cl"},
		Short:   "Clone a sandbox by creating a snapshot and spawning a new sandbox from it",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceSandboxID := args[0]

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Step 1: Connect to the source sandbox
			fmt.Printf("Connecting to source sandbox %s...\n", sourceSandboxID)
			sourceSbx, err := client.ConnectSandbox(ctx, sourceSandboxID)
			if err != nil {
				return fmt.Errorf("failed to connect to source sandbox: %w", err)
			}

			// Step 2: Create a snapshot from the source sandbox
			fmt.Println("Creating snapshot...")
			snapshot, err := sourceSbx.CreateSnapshot(ctx)
			if err != nil {
				return fmt.Errorf("failed to create snapshot: %w", err)
			}
			fmt.Printf("Snapshot created: %s\n", snapshot.SnapshotID)

			// Step 3: Create a new sandbox using the snapshot as template
			fmt.Println("Creating new sandbox from snapshot...")
			opts := []sdk.SandboxOption{sdk.WithTemplate(snapshot.SnapshotID)}
			if timeout > 0 {
				opts = append(opts, sdk.WithTimeout(timeout))
			}

			newSbx, err := client.CreateSandbox(ctx, opts...)
			if err != nil {
				return fmt.Errorf("failed to create sandbox from snapshot: %w", err)
			}

			fmt.Printf("Sandbox cloned: %s (from snapshot: %s)\n", newSbx.ID, snapshot.SnapshotID)

			if detach {
				return nil
			}
			return connectTerminal(ctx, newSbx)
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 0, "Sandbox timeout in seconds")
	cmd.Flags().BoolVar(&detach, "detach", false, "Do not connect to the sandbox after creation")
	return cmd
}
