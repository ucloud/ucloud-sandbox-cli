package sandbox

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type getOperation struct{}

func (o *getOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "get <sandbox-id>",
		Aliases: []string{"info"},
		Short:   "Show a sandbox",
		Args:    cobra.ExactArgs(1),
	}
}

func (o *getOperation) Run(ctx cmd.OperationContext) error {
	detail, err := ctx.Client.Sandboxes().Get(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	return cmd.ShowJSON(detail)
}
