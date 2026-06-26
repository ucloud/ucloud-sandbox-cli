package sandbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

func newPauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pause <sandbox-id>",
		Short: "Pause a sandbox",
		Args:  cobra.ExactArgs(1),
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
			if err := client.PauseSandbox(ctx, args[0]); err != nil {
				return err
			}
			fmt.Printf("Sandbox %s paused.\n", args[0])
			return nil
		},
	}
}
