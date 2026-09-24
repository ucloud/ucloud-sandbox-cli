package secret

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

// Command returns the root secret command group.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "secret",
		Aliases: []string{"sec"},
		Short:   "Manage secrets",
	}
	c.AddCommand(cmd.Build(&createOperation{}))
	c.AddCommand(cmd.Build(&deleteOperation{}))
	c.AddCommand(cmd.Build(&getOperation{}))
	c.AddCommand(cmd.Build(&listOperation{}))
	c.AddCommand(cmd.Build(&updateOperation{}))
	return c
}
