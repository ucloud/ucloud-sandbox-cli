package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
)

// NewRegionCmd creates the region command.
func NewRegionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "region",
		Short: "Switch the active region",
		RunE: func(cmd *cobra.Command, args []string) error {
			region, err := prompt.AskRegion(false)
			if err != nil {
				return err
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.Region = region
			if err := config.Save(cfg); err != nil {
				return err
			}

			fmt.Printf("Region switched to %q.\n", region)
			return nil
		},
	}
}
