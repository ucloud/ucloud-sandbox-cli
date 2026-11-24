package fs

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type uploadOperation struct {
	file string
}

func (o *uploadOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "upload <volume-id> <path>",
		Short: "Upload a file from stdin or a local file",
		Args:  cobra.ExactArgs(2),
	}
	c.Flags().StringVarP(&o.file, "file", "f", "", "Local file to upload (default: stdin)")
	return c
}

func (o *uploadOperation) Run(ctx cmd.OperationContext) error {
	content, err := open(ctx)
	if err != nil {
		return err
	}
	reader := ctx.Cmd.InOrStdin()
	if o.file != "" {
		file, err := os.Open(o.file)
		if err != nil {
			return err
		}
		defer file.Close()
		reader = file
	}
	_, err = content.WriteFile(ctx, ctx.Args[1], reader, nil)
	return err
}
