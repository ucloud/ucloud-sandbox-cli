package tag

import (
	"os"
	"time"

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
		Use:     "list <template>",
		Aliases: []string{"ls"},
		Short:   "List a template's tags",
		Args:    cobra.ExactArgs(1),
	}

	o.list.AddFlags(c)

	return c
}

// listedTag is a display-friendly view of api.TemplateTag for table rendering.
type listedTag struct {
	Tag       string    `table_field:"Tag" json:"tag"`
	BuildID   string    `table_field:"Build ID" json:"build_id"`
	CreatedAt time.Time `table_field:"Created" json:"created_at"`
}

func toListedTag(t api.TemplateTag) listedTag {
	return listedTag{
		Tag:       t.Tag,
		BuildID:   t.BuildID.String(),
		CreatedAt: t.CreatedAt,
	}
}

func (o *listOperation) Run(ctx cmd.OperationContext) error {
	if err := o.list.Validate(); err != nil {
		return err
	}

	tags, err := ctx.Client.Templates().ListTags(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	rows := make([]listedTag, len(tags))
	for i, t := range tags {
		rows[i] = toListedTag(t)
	}

	page := list.FromSlice(rows, o.list.Page, o.list.Limit)

	return list.Render(os.Stdout, page, o.list, identity, "No tags found.")
}

// identity is the row conversion for a page whose items are already rows.
func identity(t listedTag) listedTag { return t }
