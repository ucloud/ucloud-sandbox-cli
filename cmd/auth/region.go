package auth

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
)

type regionOperation struct{}

func (o *regionOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "region",
		Short: "Switch the active region",
		Args:  cobra.NoArgs,
	}
}

func (o *regionOperation) Run(_ cmd.OperationContext) error {
	region, err := prompt.AskRegion(false)
	if err != nil {
		return err
	}

	// The file is read rather than the resolved config, so a value that only
	// came from the environment is not written into it.
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}

	cfg.Region = region

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Region switched to %q.\n", region)

	return nil
}
