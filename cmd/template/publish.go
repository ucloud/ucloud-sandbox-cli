package template

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
)

type publishOperation struct {
	targets

	unpublish bool
}

func (o *publishOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "publish [template...]",
		Aliases: []string{"pb"},
		Short:   "Publish or unpublish templates",
		Args:    cobra.ArbitraryArgs,
	}

	o.AddFlags(c.Flags())

	c.Flags().BoolVarP(&o.unpublish, "unpublish", "", false, "Make the templates private to the team again")

	return c
}

func (o *publishOperation) Run(ctx cmd.OperationContext) error {
	ids, _, err := o.resolve(ctx)
	if err != nil {
		return err
	}

	action := "publish"
	if o.unpublish {
		action = "unpublish"
	}

	report(action, ids)

	if !o.yes {
		// Going public is the one direction that cannot be taken back from
		// whoever already copied the template.
		if !o.unpublish {
			fmt.Println("This will make the templates visible to everyone outside your team.")
		}

		confirmed, err := prompt.Confirm(fmt.Sprintf("Do you really want to %s these templates?", action))
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	for _, id := range ids {
		names, err := ctx.Client.Templates().UpdateV2(ctx, id, !o.unpublish)
		if err != nil {
			return fmt.Errorf("failed to %s template %s: %w", action, id, err)
		}

		if !o.unpublish && len(names) > 0 {
			fmt.Printf("Template %s published as %s.\n", id, strings.Join(names, ", "))
			continue
		}

		fmt.Printf("Template %s unpublished.\n", id)
	}

	return nil
}
