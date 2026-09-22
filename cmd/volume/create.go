package volume

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"cr"},
		Short:   "Create a volume",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			volume, err := client.Volumes().Create(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("failed to create volume: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Volume created: %s\n", volume.ID)
			return nil
		},
	}

	return cmd
}
