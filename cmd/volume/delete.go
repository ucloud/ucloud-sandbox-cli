package volume

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type deleteOperation struct{}

func (o *deleteOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <volume-id...>",
		Aliases: []string{"dl", "rm"},
		Short:   "Delete one or more volumes",
		Args:    cobra.MinimumNArgs(1),
	}
}

func (o *deleteOperation) Run(ctx cmd.OperationContext) error {
	// Deleting stops at the first failure, so a broken run does not keep
	// destroying volumes.
	for _, id := range ctx.Args {
		deleted, err := ctx.Client.Volumes().Delete(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete volume %s: %w", id, err)
		}

		// A volume that is already gone satisfies the intent, so it is
		// reported rather than treated as a failure.
		if !deleted {
			fmt.Printf("Volume %s not found.\n", id)
			continue
		}

		fmt.Printf("Volume %s deleted.\n", id)
	}

	return nil
}
