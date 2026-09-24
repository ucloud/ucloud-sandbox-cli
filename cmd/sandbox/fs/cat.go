package fs

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type catOperation struct {
	session
}

func (o *catOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "cat <sandbox-id> <path>",
		Short: "Print the contents of a file",
		Args:  cobra.ExactArgs(2),
	}

	o.AddFlags(c.Flags())

	return c
}

func (o *catOperation) Run(ctx cmd.OperationContext) error {
	fs, err := o.open(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	// The file is streamed rather than read whole, so printing a large one
	// does not have to hold it in memory first.
	reader, err := fs.ReadStream(ctx, ctx.Args[1])
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)

	return err
}
