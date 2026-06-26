package snapshot

import "github.com/spf13/cobra"

// NewSnapshotCmd returns the root snapshot command group.
func NewSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "snapshot",
		Aliases: []string{"snap"},
		Short:   "Manage snapshots",
	}
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newDeleteCmd())
	return cmd
}
