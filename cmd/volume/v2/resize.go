package volume

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type resizeOperation struct {
	sizeGiB int64
}

func (o *resizeOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "resize <volume-id>",
		Short: "Expand a v2 volume (shrinking is not supported)",
		Args:  cobra.ExactArgs(1),
	}
	c.Flags().Int64Var(&o.sizeGiB, "size-gib", 0, "Target capacity in GiB")
	c.MarkFlagRequired("size-gib")
	return c
}

func (o *resizeOperation) Run(ctx cmd.OperationContext) error {
	if err := validateSize(o.sizeGiB); err != nil {
		return err
	}
	id, err := parseVolumeID(ctx.Args[0])
	if err != nil {
		return err
	}
	vol, err := ctx.Client.VolumesV2().Resize(ctx, id, o.sizeGiB)
	if err != nil {
		return err
	}
	return cmd.ShowJSON(vol)
}
