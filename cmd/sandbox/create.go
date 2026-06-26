package sandbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

func newCreateCmd() *cobra.Command {
	var timeout int
	var detach bool

	cmd := &cobra.Command{
		Use:     "create [template]",
		Aliases: []string{"cr"},
		Short:   "Create a new sandbox",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			template := "base"
			if len(args) > 0 {
				template = args[0]
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
			opts := []sdk.SandboxOption{sdk.WithTemplate(template)}
			if timeout > 0 {
				opts = append(opts, sdk.WithTimeout(timeout))
			}

			sbx, err := client.CreateSandbox(ctx, opts...)
			if err != nil {
				return err
			}

			fmt.Printf("Sandbox created: %s (template: %s)\n", sbx.ID, template)

			if detach {
				return nil
			}
			return connectTerminal(ctx, sbx)
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 0, "Sandbox timeout in seconds")
	cmd.Flags().BoolVar(&detach, "detach", false, "Do not connect to the sandbox after creation")
	return cmd
}
