package volume

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <volume-id...>",
		Aliases: []string{"dl"},
		Short:   "Delete one or more volumes",
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

			for _, id := range args {
				deleted, err := client.Volumes().Delete(cmd.Context(), id)
				if err != nil {
					return fmt.Errorf("failed to delete volume %s: %w", id, err)
				}
				if !deleted {
					fmt.Fprintf(cmd.OutOrStdout(), "Volume not found: %s\n", id)
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Deleted volume: %s\n", id)
			}

			return nil
		},
	}

	return cmd
}
