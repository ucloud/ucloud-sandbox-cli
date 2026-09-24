package fs

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type mkdirOperation struct {
	session
}

func (o *mkdirOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "mkdir <sandbox-id> <dir>",
		Short: "Create a directory",
		Args:  cobra.ExactArgs(2),
	}

	o.AddFlags(c.Flags())

	return c
}

func (o *mkdirOperation) Run(ctx cmd.OperationContext) error {
	fs, err := o.open(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	dir := ctx.Args[1]

	created, err := fs.MakeDir(ctx, dir)
	if err != nil {
		return err
	}

	// A directory that is already there satisfies the intent, so it is
	// reported rather than treated as a failure.
	if !created {
		fmt.Printf("Directory already exists: %s\n", dir)
		return nil
	}

	fmt.Printf("Created directory %s\n", dir)

	return nil
}
