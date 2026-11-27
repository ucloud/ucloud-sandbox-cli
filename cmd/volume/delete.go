package volume

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/api"
)

type deleteOperation struct{}

func (o *deleteOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <volume-id...>",
		Aliases: []string{"dl", "rm"},
		Short:   "Delete one or more volumes",
		Args:    cobra.MinimumNArgs(1),
	}
}

func (o *deleteOperation) Run(ctx cmd.OperationContext) error {
	// Validate every ID before deleting any volumes.
	ids := make([]api.VolumeID, len(ctx.Args))
	for i, value := range ctx.Args {
		id, err := parseVolumeID(value)
		if err != nil {
			return err
		}
		ids[i] = id
	}
	// Deleting stops at the first failure, so a broken run does not keep
	// destroying volumes.
	for _, id := range ids {
		deleted, err := ctx.Client.Volumes().Delete(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete volume %s: %w", id, err)
		}

		// A volume that is already gone satisfies the intent, so it is
		// reported rather than treated as a failure.
		if !deleted {
			if _, err := fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Volume %s not found.\n", id); err != nil {
				return err
			}
			continue
		}

		if _, err := fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Volume %s deleted.\n", id); err != nil {
			return err
		}
	}

	return nil
}
