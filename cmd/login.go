package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
)

// NewLoginCmd creates the login command.
func NewLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with UCloud Sandbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := prompt.AskAPIKey()
			if err != nil {
				return err
			}

			region, err := prompt.AskRegion(true)
			if err != nil {
				return err
			}

			cfg := &config.Config{
				APIKey: apiKey,
				Region: region,
			}
			if err := config.Save(cfg); err != nil {
				return err
			}

			fmt.Println("Logged in successfully.")
			return nil
		},
	}
}
