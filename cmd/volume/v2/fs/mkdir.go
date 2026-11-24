package fs

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type mkdirOperation struct{}

func (o *mkdirOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "mkdir <volume-id> <path>",
		Short: "Create a directory in a v2 volume",
		Args:  cobra.ExactArgs(2),
	}
}

func (o *mkdirOperation) Run(ctx cmd.OperationContext) error {
	content, err := open(ctx)
	if err != nil {
		return err
	}
	_, err = content.MakeDir(ctx, ctx.Args[1], nil)
	return err
}
