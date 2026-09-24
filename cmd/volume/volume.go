package volume

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

// Command returns the root volume command group.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "volume",
		Aliases: []string{"vol"},
		Short:   "Manage volumes",
	}
	c.AddCommand(cmd.Build(&createOperation{}))
	c.AddCommand(cmd.Build(&deleteOperation{}))
	c.AddCommand(cmd.Build(&getOperation{}))
	c.AddCommand(cmd.Build(&listOperation{}))
	return c
}
