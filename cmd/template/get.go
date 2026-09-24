package template

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

type getOperation struct {
	params api.TemplateGetParams
}

func (o *getOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "get <template-id>",
		Aliases: []string{"show", "info"},
		Short:   "Show a template with its first page of builds",
		Args:    cobra.ExactArgs(1),
	}

	flags.NullableInt32VarP(c.Flags(), &o.params.Limit, "limit", "l", "Maximum number of builds to return")

	return c
}

func (o *getOperation) Run(ctx cmd.OperationContext) error {
	// Only the first page of builds: the endpoint returns the cursor for the
	// next one in a header the SDK does not surface, so there is nothing to
	// page with. --limit is how you widen the page.
	tpl, err := ctx.Client.Templates().Get(ctx, ctx.Args[0], &o.params)
	if err != nil {
		return err
	}

	return cmd.ShowJSON(tpl)
}
