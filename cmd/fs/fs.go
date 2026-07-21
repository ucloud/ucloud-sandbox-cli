package fs

import "github.com/spf13/cobra"

// NewFsCmd returns the root fs command group.
func NewFsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fs",
		Short: "Manage sandbox filesystem",
	}
	cmd.AddCommand(newLsCmd())
	cmd.AddCommand(newRmCmd())
	cmd.AddCommand(newMvCmd())
	cmd.AddCommand(newCpCmd())
	cmd.AddCommand(newCatCmd())
	return cmd
}
