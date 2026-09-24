package secret

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

type updateOperation struct {
	req api.SecretUpdate

	value valueSource
}

func (o *updateOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "update <name-or-id>",
		Aliases: []string{"up"},
		Short:   "Store a new version of a secret",
		Args:    cobra.ExactArgs(1),
	}

	flags.NullableStringMapVarP(c.Flags(), &o.req.Metadata, "metadata", "m", "Metadata")

	o.value.AddFlags(c.Flags())

	return c
}

func (o *updateOperation) Run(ctx cmd.OperationContext) error {
	name := ctx.Args[0]

	req := o.req

	value, err := o.value.Read("New value of " + name)
	if err != nil {
		return err
	}
	req.Value = value

	// Earlier versions are kept; this one becomes the version served to
	// readers that do not name one.
	updated, err := ctx.Client.Secrets().Update(ctx, name, req)
	if err != nil {
		return err
	}

	fmt.Printf("Secret updated: %s (version %d)\n", updated.Name, updated.CurrentVersion)

	return nil
}
