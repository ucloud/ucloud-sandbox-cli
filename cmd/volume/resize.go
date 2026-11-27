package volume

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/api"
)

type resizeOperation struct {
	sizeGiB int64
}

func (o *resizeOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "resize <volume-id>",
		Short: "Expand a volume (legacy volumes and shrinking are not supported)",
		Args:  cobra.ExactArgs(1),
	}
	c.Flags().Int64Var(&o.sizeGiB, "size-gib", 0, "New total capacity in whole GiB")
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
	vol, err := ctx.Client.Volumes().Resize(ctx, id, api.ResizeVolumeRequest{TotalSizeGiB: o.sizeGiB})
	if err != nil {
		return err
	}
	return showVolume(ctx, vol)
}
