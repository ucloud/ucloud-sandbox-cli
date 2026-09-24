package sandbox

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

type listOperation struct {
	params api.SandboxListParamsV2

	list list.Options
}

func (o *listOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List sandboxes",
		Args:    cobra.NoArgs,
	}

	sandboxListParamsVarP(c.Flags(), &o.params)

	o.list.AddFlags(c)

	return c
}

// sandboxListParamsVarP registers the flags narrowing a sandbox listing. It is
// shared by every command that picks sandboxes by their properties rather than
// by their ID -- list, and kill --all.
func sandboxListParamsVarP(fs *pflag.FlagSet, params *api.SandboxListParamsV2) {
	flags.NullableStringVarP(fs, &params.Metadata, "metadata", "m",
		"Keep only sandboxes matching this URL-encoded metadata query (e.g. user=abc&app=prod)")
	flags.NullableStringSliceVarP(fs, &params.State, "state", "s",
		"Keep only sandboxes in this state (running, paused)")
	flags.NullableStringVarP(fs, &params.Order, "order", "o",
		"Sort direction by start time (asc, desc); newest first by default")
	flags.NullableDatetimeVarP(fs, &params.StartedAfter, "started-after", "",
		`Keep only sandboxes started at or after this time ("15:04", "06-23 15:04" or "2026-07-23 15:04")`)
	flags.NullableStringVarP(fs, &params.Template, "template", "t",
		"Keep only sandboxes created from this template ID or alias")
}

// listedSandbox is a display-friendly view of api.ListedSandbox for table
// rendering.
type listedSandbox struct {
	SandboxID  string    `table_field:"Sandbox ID"`
	State      string    `table_field:"State"`
	TemplateID string    `table_field:"Template"`
	Alias      string    `table_field:"Alias"`
	StartedAt  time.Time `table_field:"Started"`
	EndAt      time.Time `table_field:"Ends"`
	CPUCount   int32     `table_field:"vCPU"`
	MemoryMB   int32     `table_field:"RAM (MB)"`
}

func toListedSandbox(s api.ListedSandbox) listedSandbox {
	row := listedSandbox{
		SandboxID:  s.SandboxID,
		State:      string(s.State),
		TemplateID: s.TemplateID,
		StartedAt:  s.StartedAt,
		EndAt:      s.EndAt,
		CPUCount:   s.CpuCount,
		MemoryMB:   s.MemoryMB,
	}

	if s.Alias != nil {
		row.Alias = *s.Alias
	}

	return row
}

func (o *listOperation) Run(ctx cmd.OperationContext) error {
	if err := o.list.Validate(); err != nil {
		return err
	}

	paginator := ctx.Client.Sandboxes().ListV2(ctx, &o.params)

	page, err := list.FromPaginator(ctx, paginator, o.list.Page, o.list.Limit)
	if err != nil {
		return err
	}

	return list.Render(os.Stdout, page, o.list, toListedSandbox, "No sandboxes found.")
}
