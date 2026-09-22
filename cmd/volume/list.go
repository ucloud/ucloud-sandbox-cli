package volume

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/table"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume"
)

// listedVolume is a display-friendly view of VolumeInfo for table rendering.
type listedVolume struct {
	VolumeID string `table_field:"Volume ID"`
	Name     string `table_field:"Name"`
}

func toListedVolume(v volume.Info) listedVolume {
	return listedVolume{
		VolumeID: v.VolumeID,
		Name:     v.Name,
	}
}

func newListCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List volumes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			volumes, err := client.Volumes().List(cmd.Context())
			if err != nil {
				return err
			}

			if format == "json" {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(volumes)
			}

			if len(volumes) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No volumes found.")
				return nil
			}

			rows := make([]listedVolume, len(volumes))
			for i, volume := range volumes {
				rows[i] = toListedVolume(volume)
			}

			out, err := table.Render(rows, 1, 0, int64(len(rows)))
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "pretty", "Output format (pretty, json)")
	return cmd
}
