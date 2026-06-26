package snapshot

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <snapshot-id...>",
		Aliases: []string{"dl"},
		Short:   "Delete one or more snapshots",
		Args:    cobra.MinimumNArgs(1),
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

			// Delete each snapshot sequentially
			for _, id := range args {
				deleteOne(ctx, client, id)
			}

			return nil
		},
	}

	return cmd
}

func deleteOne(ctx context.Context, client *sdk.Client, id string) {
	err := client.DeleteSnapshot(ctx, id)
	if err != nil {
		fmt.Printf("Failed to delete snapshot %s: %v\n", id, err)
		return
	}
	fmt.Printf("Deleted snapshot: %s\n", id)
}
