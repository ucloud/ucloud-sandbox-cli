package benchmark

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

// Command returns the root benchmark command group.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "benchmark",
		Aliases: []string{"bench"},
		Short:   "Benchmark the sandbox service",
	}
	c.AddCommand(cmd.Build(&createOperation{}))
	c.AddCommand(cmd.Build(&lifecycleOperation{}))
	return c
}
