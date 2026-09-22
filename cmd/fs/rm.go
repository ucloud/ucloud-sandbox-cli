package fs

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <sandbox-id> <path>",
		Short: "Remove a file",
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
			sbx, err := client.Sandboxes().Connect(ctx, args[0], sandbox.ConnectOptions{})
			if err != nil {
				return err
			}

			if err := sbx.Files.Remove(ctx, args[1], sandbox.FileOptions{}); err != nil {
				return err
			}

			fmt.Printf("Removed %s\n", args[1])
			return nil
		},
	}
}
