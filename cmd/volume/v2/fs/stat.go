package fs

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type statOperation struct{}

func (o *statOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "stat <volume-id> <path>",
		Short: "Show file or directory metadata as JSON",
		Args:  cobra.ExactArgs(2),
	}
}

func (o *statOperation) Run(ctx cmd.OperationContext) error {
	content, err := open(ctx)
	if err != nil {
		return err
	}
	entry, err := content.Stat(ctx, ctx.Args[1])
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(ctx.Cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(entry)
}
