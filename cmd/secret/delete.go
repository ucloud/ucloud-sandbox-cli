package secret

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type deleteOperation struct{}

func (o *deleteOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name-or-id...>",
		Aliases: []string{"dl", "rm"},
		Short:   "Delete one or more secrets",
		Args:    cobra.MinimumNArgs(1),
	}
}

func (o *deleteOperation) Run(ctx cmd.OperationContext) error {
	// Deleting stops at the first failure, so a broken run does not keep
	// revoking secrets.
	for _, name := range ctx.Args {
		deleted, err := ctx.Client.Secrets().Delete(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to delete secret %s: %w", name, err)
		}

		// A secret that is already gone satisfies the intent, so it is
		// reported rather than treated as a failure.
		if !deleted {
			fmt.Printf("Secret %s not found.\n", name)
			continue
		}

		// Every version of it is scheduled for cleanup.
		fmt.Printf("Secret %s deleted.\n", name)
	}

	return nil
}
