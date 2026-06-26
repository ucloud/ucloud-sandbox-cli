package sandbox

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

func newHostCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "host <sandbox-id> <port>",
		Short: "Print the host URL for a sandbox port",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid port %q: %w", args[1], err)
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

			fmt.Println(sbx.GetHost(port))
			return nil
		},
	}
}
