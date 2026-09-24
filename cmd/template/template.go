package template

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/template/tag"
)

// Command returns the root template command group.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "template",
		Aliases: []string{"tpl"},
		Short:   "Manage the templates sandboxes boot from",
	}
	c.AddCommand(cmd.Build(&buildOperation{}))
	c.AddCommand(cmd.Build(&deleteOperation{}))
	c.AddCommand(cmd.Build(&getOperation{}))
	c.AddCommand(cmd.Build(&initOperation{}))
	c.AddCommand(cmd.Build(&listOperation{}))
	c.AddCommand(cmd.Build(&logsOperation{}))
	c.AddCommand(cmd.Build(&publishOperation{}))
	c.AddCommand(tag.Command())
	return c
}
