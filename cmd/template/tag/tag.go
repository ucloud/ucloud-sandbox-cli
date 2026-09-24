package tag

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

// Command returns the tag command group, a subcommand of template.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "tag",
		Aliases: []string{"tags"},
		Short:   "Manage the tags pointing at a template's builds",
	}
	c.AddCommand(cmd.Build(&assignOperation{}))
	c.AddCommand(cmd.Build(&listOperation{}))
	c.AddCommand(cmd.Build(&removeOperation{}))
	return c
}
