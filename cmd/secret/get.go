package secret

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type getOperation struct{}

func (o *getOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "get <name-or-id>",
		Aliases: []string{"info"},
		Short:   "Show a secret's metadata",
		Args:    cobra.ExactArgs(1),
	}
}

func (o *getOperation) Run(ctx cmd.OperationContext) error {
	// The endpoint returns the secret's metadata only: its value is
	// write-only and never leaves the platform.
	sec, err := ctx.Client.Secrets().Get(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	return cmd.ShowJSON(sec)
}
