package template

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
)

type deleteOperation struct {
	targets
}

func (o *deleteOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "delete [template...]",
		Aliases: []string{"dl", "rm"},
		Short:   "Delete one or more templates",
		Args:    cobra.ArbitraryArgs,
	}

	o.AddFlags(c.Flags())

	return c
}

func (o *deleteOperation) Run(ctx cmd.OperationContext) error {
	ids, local, err := o.resolve(ctx)
	if err != nil {
		return err
	}

	report("delete", ids)

	if !o.yes {
		confirmed, err := prompt.Confirm("Do you really want to delete these templates?")
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	// Deleting stops at the first failure, so a broken run does not keep
	// destroying templates.
	total := 0
	for _, id := range ids {
		deleted, err := ctx.Client.Templates().Delete(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete template %s: %w", id, err)
		}

		// A template that is already gone satisfies the intent.
		if !deleted {
			fmt.Printf("Template %s not found.\n", id)
			continue
		}

		total++
		fmt.Printf("Template %s deleted.\n", id)
	}

	// The local config points at a template that no longer exists, so it goes
	// with it -- but only when it is what named the template.
	if local != nil {
		if err := deleteConfig(o.path); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	fmt.Printf("Deleted %d template(s).\n", total)

	return nil
}
