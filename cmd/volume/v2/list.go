package volume

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/v2/api"
)

type listOperation struct {
	list list.Options
}

func (o *listOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List v2 volumes",
		Args:    cobra.NoArgs,
	}
	o.list.AddFlags(c)
	return c
}

type listedVolume struct {
	VolumeID  string    `table_field:"Volume ID"`
	Name      string    `table_field:"Name"`
	SizeGiB   int64     `table_field:"Size (GiB)"`
	MountIP   string    `table_field:"Mount IP"`
	CreatedAt time.Time `table_field:"Created"`
}

func toListedVolume(v api.Volume) listedVolume {
	return listedVolume{
		VolumeID: v.VolumeID.String(), Name: v.Name, SizeGiB: v.SizeGiB,
		MountIP: v.MountIP, CreatedAt: v.CreatedAt,
	}
}

func (o *listOperation) Run(ctx cmd.OperationContext) error {
	if err := o.list.Validate(); err != nil {
		return err
	}
	page, err := o.fetchPage(ctx)
	if err != nil {
		return err
	}
	return list.Render(os.Stdout, page, o.list, toListedVolume, "No volumes found.")
}

// The API accepts at most 100 items per request. Fetch enough batches for the
// CLI page, or every batch when --limit is zero.
func (o *listOperation) fetchPage(ctx cmd.OperationContext) (list.Page[api.Volume], error) {
	if o.list.Limit > 0 && o.list.Page-1 > math.MaxInt/o.list.Limit {
		return list.Page[api.Volume]{}, fmt.Errorf("--page and --limit are too large")
	}
	offset := (o.list.Page - 1) * o.list.Limit
	var page list.Page[api.Volume]
	for {
		limit := 100
		if o.list.Limit > 0 {
			limit = min(limit, o.list.Limit-len(page.Items))
		}
		batch, err := ctx.Client.VolumesV2().List(ctx, &api.ListVolumesParams{Offset: &offset, Limit: &limit})
		if err != nil {
			return list.Page[api.Volume]{}, err
		}
		page.Items = append(page.Items, batch.Items...)
		page.Total = batch.Total
		offset += len(batch.Items)
		page.HasMore = int64(offset) < page.Total
		if !page.HasMore || len(batch.Items) == 0 || (o.list.Limit > 0 && len(page.Items) >= o.list.Limit) {
			return page, nil
		}
	}
}
