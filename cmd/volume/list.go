package volume

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/api"
)

type listOperation struct {
	list list.Options
}

func (o *listOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List volumes",
		Args:    cobra.NoArgs,
	}

	o.list.AddFlags(c)

	return c
}

// listedVolume is a display-friendly view of api.Volume for table rendering.
type listedVolume struct {
	VolumeID         string `table_field:"Volume ID"`
	Name             string `table_field:"Name"`
	TotalSizeGiB     string `table_field:"Total (GiB)"`
	UsedSizeGiB      string `table_field:"Used (GiB)"`
	AvailableSizeGiB string `table_field:"Available (GiB)"`
}

func toListedVolume(v api.Volume) listedVolume {
	row := listedVolume{
		VolumeID: v.VolumeID.String(), Name: v.Name,
		TotalSizeGiB: "-", UsedSizeGiB: "-", AvailableSizeGiB: "-",
	}
	if v.TotalSizeGiB != nil {
		row.TotalSizeGiB = strconv.FormatInt(*v.TotalSizeGiB, 10)
	}
	if v.UsedSizeGiB != nil {
		row.UsedSizeGiB = strconv.FormatFloat(*v.UsedSizeGiB, 'f', -1, 64)
	}
	if v.AvailableSizeGiB != nil {
		row.AvailableSizeGiB = strconv.FormatFloat(*v.AvailableSizeGiB, 'f', -1, 64)
	}
	return row
}

func (o *listOperation) Run(ctx cmd.OperationContext) error {
	if err := o.list.Validate(); err != nil {
		return err
	}

	// The endpoint is not paginated, so the whole listing arrives at once and
	// is paged client-side.
	volumes, err := ctx.Client.Volumes().List(ctx)
	if err != nil {
		return err
	}

	page := list.FromSlice(volumes, o.list.Page, o.list.Limit)

	return list.Render(ctx.Cmd.OutOrStdout(), page, o.list, toListedVolume, "No volumes found.")
}
