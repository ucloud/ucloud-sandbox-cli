package fs

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

func newCatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cat <sandbox-id> <path>",
		Short: "Print the contents of a file",
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

			rc, err := sbx.Files.ReadStream(ctx, args[1])
			if err != nil {
				return err
			}
			defer rc.Close()

			_, err = io.Copy(os.Stdout, rc)
			return err
		},
	}
}
