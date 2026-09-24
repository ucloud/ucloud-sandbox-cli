package secret

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

type listOperation struct {
	params api.SecretListParams

	list list.Options
}

func (o *listOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List secrets",
		Args:    cobra.NoArgs,
	}

	// The params' own cursor and page size are not exposed: paging is what
	// --page and --limit do, and two ways of asking for it would conflict.
	o.list.AddFlags(c)

	return c
}

// listedSecret is a display-friendly view of api.Secret for table rendering.
type listedSecret struct {
	SecretID       string    `table_field:"Secret ID"`
	Name           string    `table_field:"Name"`
	CurrentVersion int64     `table_field:"Version"`
	CreatedAt      time.Time `table_field:"Created"`
	UpdatedAt      time.Time `table_field:"Updated"`
}

func toListedSecret(s api.Secret) listedSecret {
	return listedSecret{
		SecretID:       s.SecretID,
		Name:           s.Name,
		CurrentVersion: s.CurrentVersion,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

func (o *listOperation) Run(ctx cmd.OperationContext) error {
	if err := o.list.Validate(); err != nil {
		return err
	}

	paginator := ctx.Client.Secrets().List(ctx, &o.params)

	page, err := list.FromPaginator(ctx, paginator, o.list.Page, o.list.Limit)
	if err != nil {
		return err
	}

	return list.Render(os.Stdout, page, o.list, toListedSecret, "No secrets found.")
}
