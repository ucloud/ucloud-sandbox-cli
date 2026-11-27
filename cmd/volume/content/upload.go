package content

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	contentapi "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/content/api"
)

type uploadOperation struct {
	file  string
	force *bool
}

func (o *uploadOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "upload <volume-id> <path>",
		Short: "Upload a file from stdin or a local file",
		Args:  cobra.ExactArgs(2),
	}
	c.Flags().StringVarP(&o.file, "file", "f", "", "Local file to upload (default: stdin)")
	flags.NullableBoolVarP(c.Flags(), &o.force, "force", "", "Create missing parents and overwrite an existing regular file")
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
	_, err = content.WriteFile(ctx, &contentapi.WriteVolumeFileParams{Path: ctx.Args[1], Force: o.force}, reader)
	return err
}
