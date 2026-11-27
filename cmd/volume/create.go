package volume

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/api"
)

type createOperation struct {
	req api.NewVolume
}

func (o *createOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"cr"},
		Short:   "Create a volume",
		Args:    cobra.ExactArgs(1),
	}
	flags.NullableInt64VarP(c.Flags(), &o.req.TotalSizeGiB, "size-gib", "", "Capacity in whole GiB (server default: 1)")
	return c
}

func (o *createOperation) Run(ctx cmd.OperationContext) error {
	o.req.Name = strings.TrimSpace(ctx.Args[0])
	if o.req.Name == "" {
		return fmt.Errorf("volume name must not be empty")
	}
	if o.req.TotalSizeGiB != nil {
		if err := validateSize(*o.req.TotalSizeGiB); err != nil {
			return err
		}
	}
	vol, err := ctx.Client.Volumes().Create(ctx, o.req)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Volume created: %s\n", vol.VolumeID)
	return err
}
