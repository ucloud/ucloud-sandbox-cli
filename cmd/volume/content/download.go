package content

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	contentapi "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/content/api"
)

type downloadOperation struct {
	file string
}

func (o *downloadOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "download <volume-id> <path>",
		Short: "Download a file to stdout or a local file",
		Args:  cobra.ExactArgs(2),
	}
	c.Flags().StringVarP(&o.file, "file", "f", "", "Local destination file (default: stdout)")
	return c
}

func (o *downloadOperation) Run(ctx cmd.OperationContext) error {
	content, err := open(ctx)
	if err != nil {
		return err
	}
	response, err := content.ReadFile(ctx, &contentapi.ReadVolumeFileParams{Path: ctx.Args[1]})
	if err != nil {
		return err
	}
	defer response.Body.Close()
	reader := response.Body

	if o.file == "" {
		_, err = io.Copy(ctx.Cmd.OutOrStdout(), reader)
		return err
	}
	file, err := os.Create(o.file)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, reader)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
