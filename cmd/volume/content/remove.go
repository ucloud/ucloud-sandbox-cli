package content

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	contentapi "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/content/api"
)

type removeOperation struct{}

func (o *removeOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <volume-id> <path>",
		Aliases: []string{"rm"},
		Short:   "Recursively remove a file or directory",
		Args:    cobra.ExactArgs(2),
	}
}

func (o *removeOperation) Run(ctx cmd.OperationContext) error {
	content, err := open(ctx)
	if err != nil {
		return err
	}
	return content.Remove(ctx, &contentapi.DeleteVolumePathParams{Path: ctx.Args[1]})
}
