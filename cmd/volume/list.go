package volume

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
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
	VolumeID string `table_field:"Volume ID"`
	Name     string `table_field:"Name"`
}

func toListedVolume(v api.Volume) listedVolume {
	return listedVolume{
		VolumeID: v.VolumeID,
		Name:     v.Name,
	}
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

	return list.Render(os.Stdout, page, o.list, toListedVolume, "No volumes found.")
}
