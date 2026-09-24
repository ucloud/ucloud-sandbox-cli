package sandbox

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/sandbox/fs"
)

// NewSandboxCmd returns the root sandbox command group.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "sandbox",
		Aliases: []string{"sbx"},
		Short:   "Manage sandboxes",
	}
	c.AddCommand(cmd.Build(&connectOperation{}))
	c.AddCommand(cmd.Build(&createOperation{}))
	c.AddCommand(cmd.Build(&execOperation{}))
	c.AddCommand(cmd.Build(&forkOperation{}))
	c.AddCommand(fs.Command())
	c.AddCommand(cmd.Build(&getOperation{}))
	c.AddCommand(cmd.Build(&hostOperation{}))
	c.AddCommand(cmd.Build(&killOperation{}))
	c.AddCommand(cmd.Build(&listOperation{}))
	c.AddCommand(cmd.Build(&logsOperation{}))
	c.AddCommand(cmd.Build(&metricsOperation{}))
	c.AddCommand(cmd.Build(&pauseOperation{}))
	return c
}
