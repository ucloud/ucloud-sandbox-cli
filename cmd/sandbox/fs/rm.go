package fs

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type rmOperation struct {
	session
}

func (o *rmOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "rm <sandbox-id> <path>",
		Short: "Remove a file or directory",
		Args:  cobra.ExactArgs(2),
	}

	o.AddFlags(c.Flags())

	return c
}

func (o *rmOperation) Run(ctx cmd.OperationContext) error {
	fs, err := o.open(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	path := ctx.Args[1]

	// A directory goes with everything under it.
	if err := fs.Remove(ctx, path); err != nil {
		return err
	}

	fmt.Printf("Removed %s\n", path)

	return nil
}
