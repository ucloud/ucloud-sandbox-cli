package tag

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
)

type removeOperation struct {
	yes bool
}

func (o *removeOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "remove <template> <tag> [tag...]",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove tags from a template",
		Args:    cobra.MinimumNArgs(2),
	}

	c.Flags().BoolVarP(&o.yes, "yes", "y", false, "Don't ask for confirmation")

	return c
}

func (o *removeOperation) Run(ctx cmd.OperationContext) error {
	name, tags := ctx.Args[0], ctx.Args[1:]

	if !o.yes {
		confirmed, err := prompt.Confirm(
			fmt.Sprintf("Remove tags %s from template %s?", strings.Join(tags, ", "), name))
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	if err := ctx.Client.Templates().DeleteTags(ctx, name, tags); err != nil {
		return err
	}

	fmt.Printf("Removed tags %s from template %s.\n", strings.Join(tags, ", "), name)

	return nil
}
