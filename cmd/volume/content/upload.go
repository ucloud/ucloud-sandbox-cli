package content

import (
	"fmt"
	"os"

	"github.com/schollz/progressbar/v3"
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
	var progress *progressbar.ProgressBar
	if o.file != "" {
		file, err := os.Open(o.file)
		if err != nil {
			return err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("%s: cannot upload a directory", o.file)
		}
		progress = newTransferProgress(ctx.Cmd.ErrOrStderr(), info.Size(), "Uploading")
		defer func() {
			progress.Exit()
			fmt.Fprintln(ctx.Cmd.ErrOrStderr())
		}()
		tracked := progressbar.NewReader(file, progress)
		reader = &tracked
	}
	_, err = content.WriteFile(ctx, &contentapi.WriteVolumeFileParams{Path: ctx.Args[1], Force: o.force}, reader)
	if err == nil && progress != nil {
		return progress.Finish()
	}
	return err
}
