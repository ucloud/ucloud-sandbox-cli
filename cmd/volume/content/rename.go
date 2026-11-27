package content

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	contentapi "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/content/api"
)

type renameOperation struct{}

func (o *renameOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "rename <volume-id> <source-path> <destination-path>",
		Aliases: []string{"mv"},
		Short:   "Move or rename a file or directory within a volume",
		Args:    cobra.ExactArgs(3),
	}
}

func (o *renameOperation) Run(ctx cmd.OperationContext) error {
	content, err := open(ctx)
	if err != nil {
		return err
	}
	_, err = content.Rename(ctx, contentapi.RenamePath{
		SourcePath: ctx.Args[1], DestinationPath: ctx.Args[2],
	})
	return err
}
