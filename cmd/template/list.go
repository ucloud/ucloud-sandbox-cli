package template

import (
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

type listOperation struct {
	params api.TemplateListParamsV2

	list list.Options
}

func (o *listOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List templates",
		Args:    cobra.NoArgs,
	}

	o.list.AddFlags(c)

	return c
}

// listedTemplate is a display-friendly view of api.Template for table
// rendering.
type listedTemplate struct {
	TemplateID  string    `table_field:"Template ID" json:"template_id"`
	Names       string    `table_field:"Name" json:"names"`
	EnvdVersion string    `table_field:"Envd Version" json:"envd_version"`
	Status      string    `table_field:"Status" json:"status"`
	Visibility  string    `table_field:"Access" json:"access"`
	CPUCount    int32     `table_field:"vCPU" json:"cpu_count"`
	MemoryMB    int32     `table_field:"RAM (MB)" json:"memory_mb"`
	CreatedAt   time.Time `table_field:"Created" json:"created_at"`
}

func toListedTemplate(t api.Template) listedTemplate {
	status := string(t.BuildStatus)
	if status == "" {
		status = "-"
	}

	visibility := "Private"
	if t.Public {
		visibility = "Public"
	}

	return listedTemplate{
		TemplateID:  t.TemplateID,
		Names:       strings.Join(t.Names, ", "),
		EnvdVersion: t.EnvdVersion,
		Status:      status,
		Visibility:  visibility,
		CPUCount:    t.CpuCount,
		MemoryMB:    t.MemoryMB,
		CreatedAt:   t.CreatedAt,
	}
}

func (o *listOperation) Run(ctx cmd.OperationContext) error {
	if err := o.list.Validate(); err != nil {
		return err
	}

	paginator := ctx.Client.Templates().ListV2(ctx, &o.params)

	page, err := list.FromPaginator(ctx, paginator, o.list.Page, o.list.Limit)
	if err != nil {
		return err
	}

	return list.Render(os.Stdout, page, o.list, toListedTemplate, "No templates found.")
}
