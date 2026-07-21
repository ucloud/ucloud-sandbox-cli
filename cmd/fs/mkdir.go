package fs

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

func newMkdirCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mkdir <sandbox-id> <dir>",
		Short: "Create a directory",
		Args:  cobra.ExactArgs(2),
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

			created, err := sbx.Files.MakeDir(ctx, args[1])
			if err != nil {
				return err
			}
			if !created {
				fmt.Printf("Directory already exists: %s\n", args[1])
				return nil
			}

			fmt.Printf("Created directory %s\n", args[1])
			return nil
		},
	}
}
