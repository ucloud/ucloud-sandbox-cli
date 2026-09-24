package fs

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type mvOperation struct {
	session
}

func (o *mvOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "mv <sandbox-id> <old-path> <new-path>",
		Short: "Move a file",
		Args:  cobra.ExactArgs(3),
	}

	o.AddFlags(c.Flags())

	return c
}

func (o *mvOperation) Run(ctx cmd.OperationContext) error {
	fs, err := o.open(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	oldPath := ctx.Args[1]

	// The entry comes back at its new path, which is what is reported: envd
	// may have resolved the destination differently from what was asked for.
	info, err := fs.Rename(ctx, oldPath, ctx.Args[2])
	if err != nil {
		return err
	}

	fmt.Printf("Moved %s -> %s\n", oldPath, info.GetPath())

	return nil
}
