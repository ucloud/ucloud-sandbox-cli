package template

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
)

func newDeleteCmd() *cobra.Command {
	var path string
	var yes bool
	var selectMode bool

	cmd := &cobra.Command{
		Use:     "delete [template...]",
		Aliases: []string{"dl"},
		Short:   "Delete sandbox templates",
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

			// Resolve target templates
			targets, localCfg, err := resolveTargets(ctx, client, args, path, selectMode)
			if err != nil {
				return err
			}

			if len(targets) == 0 {
				fmt.Println("No templates selected.")
				return nil
			}

			// Print targets
			fmt.Println("\nTemplates to delete:")
			for _, id := range targets {
				fmt.Printf("  - %s\n", id)
			}
			fmt.Println()

			// Confirm
			if !yes {
				confirmed, err := prompt.Confirm("Do you really want to delete these templates?")
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Println("Canceled.")
					return nil
				}
			}

			// Delete each
			for _, id := range targets {
				fmt.Printf("Deleting template %s...", id)
				if err := client.DeleteTemplate(ctx, id); err != nil {
					fmt.Printf(" failed: %v\n", err)
					continue
				}
				fmt.Println(" done")
			}

			// Delete local config if applicable
			if localCfg != nil {
				if err := deleteConfig(path); err != nil {
					fmt.Printf("Warning: failed to delete config: %v\n", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", ".", "Project root path")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	cmd.Flags().BoolVarP(&selectMode, "select", "s", false, "Interactive selection")
	return cmd
}
