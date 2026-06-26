package template

import "github.com/spf13/cobra"

// NewTemplateCmd returns the root template command group.
func NewTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"tpl"},
		Short:   "Manage sandbox templates",
	}
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newBuildCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newPublishCmd())
	cmd.AddCommand(newInitCmd())
	return cmd
}
