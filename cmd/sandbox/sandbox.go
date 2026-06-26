package sandbox

import "github.com/spf13/cobra"

// NewSandboxCmd returns the root sandbox command group.
func NewSandboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sandbox",
		Aliases: []string{"sbx"},
		Short:   "Manage sandboxes",
	}
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newCloneCmd())
	cmd.AddCommand(newConnectCmd())
	cmd.AddCommand(newKillCmd())
	cmd.AddCommand(newPauseCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newExecCmd())
	cmd.AddCommand(newHostCmd())
	cmd.AddCommand(newMetricsCmd())
	return cmd
}
