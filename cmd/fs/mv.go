package fs

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

func newMvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mv <sandbox-id> <old-path> <new-path>",
		Short: "Move a file",
		Args:  cobra.ExactArgs(3),
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
			sbx, err := client.ConnectSandbox(ctx, args[0])
			if err != nil {
				return err
			}

			info, err := sbx.Files.Rename(ctx, args[1], args[2])
			if err != nil {
				return err
			}

			fmt.Printf("Moved %s -> %s\n", args[1], info.Path)
			return nil
		},
	}
}
