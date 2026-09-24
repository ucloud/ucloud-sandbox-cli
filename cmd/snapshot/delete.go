package snapshot

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type deleteOperation struct{}

func (o *deleteOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <snapshot-id...>",
		Aliases: []string{"dl", "rm"},
		Short:   "Delete one or more snapshots",
		Args:    cobra.MinimumNArgs(1),
	}
}

func (o *deleteOperation) Run(ctx cmd.OperationContext) error {
	// Deleting stops at the first failure, so a broken run does not keep
	// destroying snapshots.
	total := 0
	for _, id := range ctx.Args {
		// This takes every build of the snapshot with it.
		deleted, err := ctx.Client.Sandboxes().DeleteSnapshot(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete snapshot %s: %w", id, err)
		}

		// A snapshot that is already gone satisfies the intent.
		if !deleted {
			fmt.Printf("Snapshot %s not found.\n", id)
			continue
		}

		total++
		fmt.Printf("Snapshot %s deleted.\n", id)
	}

	fmt.Printf("Deleted %d snapshot(s).\n", total)

	return nil
}
