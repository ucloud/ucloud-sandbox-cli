package volume

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/volume/content"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/api"
)

// Command returns the root volume command group.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "volume",
		Aliases: []string{"vol"},
		Short:   "Manage volumes",
	}
	c.AddCommand(cmd.Build(&createOperation{}))
	c.AddCommand(cmd.Build(&deleteOperation{}))
	c.AddCommand(cmd.Build(&getOperation{}))
	c.AddCommand(cmd.Build(&listOperation{}))
	c.AddCommand(cmd.Build(&resizeOperation{}))
	c.AddCommand(content.Command())
	return c
}

func parseVolumeID(value string) (api.VolumeID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return api.VolumeID{}, fmt.Errorf("invalid volume ID %q: expected UUID: %w", value, err)
	}
	return id, nil
}

func validateSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("invalid --size-gib %d: must be greater than 0", size)
	}
	return nil
}

func showVolume(ctx cmd.OperationContext, vol *api.Volume) error {
	encoder := json.NewEncoder(ctx.Cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(vol)
}
