package secret

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/secret"
)

type createOperation struct {
	req api.NewSecret

	value valueSource
}

func (o *createOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"cr"},
		Short:   "Create a secret",
		Args:    cobra.ExactArgs(1),
	}

	flags.NullableStringMapVarP(c.Flags(), &o.req.Metadata, "metadata", "m", "Metadata")

	o.value.AddFlags(c.Flags())

	return c
}

func (o *createOperation) Run(ctx cmd.OperationContext) error {
	req := o.req
	req.Name = ctx.Args[0]

	// Checking here turns what would be an opaque 400 into an error naming the
	// rule the name broke.
	if err := secret.ValidateName(req.Name); err != nil {
		return err
	}

	value, err := o.value.Read("Value of " + req.Name)
	if err != nil {
		return err
	}
	req.Value = value

	created, err := ctx.Client.Secrets().Create(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("Secret created: %s (%s)\n", created.Name, created.SecretID)

	return nil
}
