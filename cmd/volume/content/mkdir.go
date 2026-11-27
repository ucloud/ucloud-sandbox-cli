package content

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	contentapi "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/content/api"
)

type mkdirOperation struct {
	parents *bool
}

func (o *mkdirOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "mkdir <volume-id> <path>",
		Short: "Create a directory in a volume",
		Args:  cobra.ExactArgs(2),
	}
	flags.NullableBoolVarP(c.Flags(), &o.parents, "parents", "p", "Create missing parents and allow an existing directory")
	return c
}

func (o *mkdirOperation) Run(ctx cmd.OperationContext) error {
	content, err := open(ctx)
	if err != nil {
		return err
	}
	_, err = content.MakeDir(ctx, contentapi.CreateDirectory{Path: ctx.Args[1], CreateParents: o.parents})
	return err
}
