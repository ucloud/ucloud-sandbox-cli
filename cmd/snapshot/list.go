package snapshot

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

type listOperation struct {
	params api.SnapshotListParams

	list list.Options
}

func (o *listOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List snapshots",
		Args:    cobra.NoArgs,
	}

	flags.NullableStringVarP(c.Flags(), &o.params.SandboxID, "sandbox-id", "s",
		"Keep only snapshots taken of this sandbox")
	flags.NullableStringVarP(c.Flags(), &o.params.Name, "name", "n",
		"Keep only snapshots matching this name or ID, optionally tag-qualified")

	o.list.AddFlags(c)

	return c
}

// listedSnapshot is a display-friendly view of api.SnapshotInfo for table
// rendering.
type listedSnapshot struct {
	SnapshotID string `table_field:"Snapshot ID" json:"snapshot_id"`
	Names      string `table_field:"Names" json:"names"`
}

func toListedSnapshot(s api.SnapshotInfo) listedSnapshot {
	return listedSnapshot{
		SnapshotID: s.SnapshotID,
		Names:      strings.Join(s.Names, ", "),
	}
}

func (o *listOperation) Run(ctx cmd.OperationContext) error {
	if err := o.list.Validate(); err != nil {
		return err
	}

	paginator := ctx.Client.Sandboxes().ListSnapshots(ctx, &o.params)

	page, err := list.FromPaginator(ctx, paginator, o.list.Page, o.list.Limit)
	if err != nil {
		return err
	}

	return list.Render(os.Stdout, page, o.list, toListedSnapshot, "No snapshots found.")
}
