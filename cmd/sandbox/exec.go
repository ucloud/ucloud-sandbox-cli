package sandbox

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
)

func newExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec <sandbox-id> <command>",
		Short: "Execute a command in a sandbox",
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

			result, err := sbx.Commands.Run(ctx, args[1],
				sandbox.CommandOptions{
					OnStdout: func(s string) { fmt.Fprint(os.Stdout, s) },
					OnStderr: func(s string) { fmt.Fprint(os.Stderr, s) },
				},
			)
			if err != nil {
				// Print any remaining output before returning the error.
				if result != nil {
					fmt.Fprint(os.Stdout, result.Stdout)
					fmt.Fprint(os.Stderr, result.Stderr)
				}
				return err
			}
			return nil
		},
	}
}
