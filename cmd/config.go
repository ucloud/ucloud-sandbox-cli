package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	internalconfig "github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

// NewConfigCmd creates the config command.
func NewConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show the current configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internalconfig.Load()
			if err != nil {
				return err
			}

			masked := *cfg
			masked.APIKey = maskAPIKey(masked.APIKey)
			return writeConfig(cmd.OutOrStdout(), &masked)
		},
	}
}

func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	return "****"
}

func writeConfig(w io.Writer, cfg *internalconfig.Config) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("print config: %w", err)
	}
	return nil
}
