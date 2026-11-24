package fs

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	volume "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/v2"
)

// Command returns the filesystem command group for v2 volumes.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "fs",
		Short: "Manage files in v2 volumes",
	}
	c.AddCommand(cmd.Build(&uploadOperation{}))
	c.AddCommand(cmd.Build(&downloadOperation{}))
	c.AddCommand(cmd.Build(&listOperation{}))
	c.AddCommand(cmd.Build(&mkdirOperation{}))
	c.AddCommand(cmd.Build(&removeOperation{}))
	c.AddCommand(cmd.Build(&statOperation{}))
	return c
}

func open(ctx cmd.OperationContext) (*volume.Content, error) {
	id, err := uuid.Parse(ctx.Args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid volume ID %q: expected UUID: %w", ctx.Args[0], err)
	}
	if ctx.Args[1] == "" {
		return nil, fmt.Errorf("volume path must not be empty; use / for the root")
	}
	return ctx.Client.VolumeContentV2(id), nil
}
