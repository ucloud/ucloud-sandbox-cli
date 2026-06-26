package template

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
)

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func newPublishCmd() *cobra.Command {
	var path string
	var yes bool
	var selectMode bool
	var unpublish bool

	cmd := &cobra.Command{
		Use:     "publish [template...]",
		Aliases: []string{"pb"},
		Short:   "Publish or unpublish sandbox templates",
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

			// Resolve targets
			targets, _, err := resolveTargets(ctx, client, args, path, selectMode)
			if err != nil {
				return err
			}

			if len(targets) == 0 {
				fmt.Println("No templates selected.")
				return nil
			}

			action := "publish"
			if unpublish {
				action = "unpublish"
			}

			// Print targets
			fmt.Printf("\nTemplates to %s:\n", action)
			for _, id := range targets {
				fmt.Printf("  - %s\n", id)
			}
			fmt.Println()

			// Confirm
			if !yes {
				msg := fmt.Sprintf("Do you really want to %s these templates?", action)
				if !unpublish {
					msg += "\n⚠️  This will make the templates public to everyone outside your team"
				}
				confirmed, err := prompt.Confirm(msg)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Println("Canceled.")
					return nil
				}
			}

			// Publish/unpublish each
			for _, id := range targets {
				fmt.Printf("%s template %s...", capitalize(action), id)
				names, err := client.SetTemplatePublic(ctx, id, !unpublish)
				if err != nil {
					fmt.Printf(" failed: %v\n", err)
					continue
				}
				if !unpublish && len(names) > 0 {
					fmt.Printf(" done (published as: %s)\n", strings.Join(names, ", "))
				} else {
					fmt.Println(" done")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", ".", "Project root path")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	cmd.Flags().BoolVarP(&selectMode, "select", "s", false, "Interactive selection")
	cmd.Flags().BoolVar(&unpublish, "unpublish", false, "Unpublish instead of publish")
	return cmd
}
