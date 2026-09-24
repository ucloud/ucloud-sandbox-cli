package auth

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
)

type loginOperation struct{}

func (o *loginOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with UCloud Sandbox",
		Args:  cobra.NoArgs,
	}
}

func (o *loginOperation) Run(_ cmd.OperationContext) error {
	apiKey, err := prompt.AskAPIKey()
	if err != nil {
		return err
	}

	region, err := prompt.AskRegion(true)
	if err != nil {
		return err
	}

	// The file is read rather than replaced, so settings logging in says
	// nothing about -- the domain, the registry credentials -- survive it.
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}

	cfg.APIKey = apiKey

	// Skipping the region prompt leaves the configured one alone.
	if region != "" {
		cfg.Region = region
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Println("Logged in successfully.")

	return nil
}
