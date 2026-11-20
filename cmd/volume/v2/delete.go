package volume

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/v2/api"
)

type deleteOperation struct{}

func (o *deleteOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <volume-id...>",
		Aliases: []string{"dl", "rm"},
		Short:   "Delete one or more v2 volumes",
		Args:    cobra.MinimumNArgs(1),
	}
}

func (o *deleteOperation) Run(ctx cmd.OperationContext) error {
	ids := make([]api.VolumeID, len(ctx.Args))
	for i, arg := range ctx.Args {
		id, err := parseVolumeID(arg)
		if err != nil {
			return err
		}
		ids[i] = id
	}
	for _, id := range ids {
		if err := ctx.Client.VolumesV2().Delete(ctx, id); err != nil {
			return fmt.Errorf("failed to delete volume %s: %w", id, err)
		}
		fmt.Printf("Volume %s deleted.\n", id)
	}
	return nil
}
