package volume

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type getOperation struct{}

func (o *getOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "get <volume-id>",
		Aliases: []string{"info"},
		Short:   "Show a volume",
		Args:    cobra.ExactArgs(1),
	}
}

func (o *getOperation) Run(ctx cmd.OperationContext) error {
	vol, err := ctx.Client.Volumes().Get(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	return cmd.ShowJSON(vol)
}
