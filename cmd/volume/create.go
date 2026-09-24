package volume

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type createOperation struct{}

func (o *createOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"cr"},
		Short:   "Create a volume",
		Args:    cobra.ExactArgs(1),
	}
}

func (o *createOperation) Run(ctx cmd.OperationContext) error {
	// The response also carries the volume's content token, which `get`
	// returns; a created volume is reported by its ID alone.
	vol, err := ctx.Client.Volumes().Create(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Volume created: %s\n", vol.VolumeID)

	return nil
}
