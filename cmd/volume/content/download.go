package content

import (
	"fmt"
	"io"
	"os"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	contentapi "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/content/api"
	"golang.org/x/term"
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
		out := ctx.Cmd.OutOrStdout()
		if file, ok := out.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
			return writeText(out, reader)
		}
		_, err = io.Copy(out, reader)
		return err
	}
	file, err := os.Create(o.file)
	if err != nil {
		return err
	}
	progress := newTransferProgress(ctx.Cmd.ErrOrStderr(), response.ContentLength, "Downloading")
	defer func() {
		progress.Exit()
		fmt.Fprintln(ctx.Cmd.ErrOrStderr())
	}()
	tracked := progressbar.NewReader(reader, progress)
	_, err = io.Copy(file, &tracked)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return progress.Finish()
}
