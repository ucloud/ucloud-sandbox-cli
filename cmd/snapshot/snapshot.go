package snapshot

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

// Command returns the root snapshot command group.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "snapshot",
		Aliases: []string{"snap"},
		Short:   "Manage the snapshots taken of sandboxes",
	}
	c.AddCommand(cmd.Build(&createOperation{}))
	c.AddCommand(cmd.Build(&deleteOperation{}))
	c.AddCommand(cmd.Build(&listOperation{}))
	return c
}
