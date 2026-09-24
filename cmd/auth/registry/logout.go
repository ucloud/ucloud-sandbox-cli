package registry

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

type logoutOperation struct{}

func (o *logoutOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "logout [domain]",
		Short: "Remove the credentials for a container registry",
		Args:  cobra.MaximumNArgs(1),
	}
}

func (o *logoutOperation) Run(ctx cmd.OperationContext) error {
	domain, err := domainOf(ctx.Args)
	if err != nil {
		return err
	}

	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}

	// Having no credentials for the registry is what the caller asked for, not
	// a failure.
	if _, ok := cfg.Registries[domain]; !ok {
		fmt.Printf("No credentials stored for %s.\n", domain)
		return nil
	}

	delete(cfg.Registries, domain)

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Logged out of %s.\n", domain)

	return nil
}
