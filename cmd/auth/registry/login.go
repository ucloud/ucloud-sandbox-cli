package registry

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
	"github.com/ucloud/ucloud-sandbox-cli/internal/registry"
)

type loginOperation struct{}

func (o *loginOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "login [domain]",
		Short: "Store the credentials for a container registry",
		Args:  cobra.MaximumNArgs(1),
	}
}

func (o *loginOperation) Run(ctx cmd.OperationContext) error {
	domain, err := domainOf(ctx.Args)
	if err != nil {
		return err
	}

	fmt.Printf("Logging in to %s\n", domain)

	username, err := prompt.AskUsername()
	if err != nil {
		return err
	}

	// Asking rather than taking a flag keeps the password out of the shell's
	// history and out of the machine's process list.
	password, err := prompt.AskSecret("Password")
	if err != nil {
		return err
	}

	// The file is read rather than the resolved config, so credentials that
	// only came from the environment are not written into it.
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}

	if cfg.Registries == nil {
		cfg.Registries = map[string]config.RegistryAuth{}
	}
	cfg.Registries[domain] = config.RegistryAuth{Username: username, Password: password}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Logged in to %s as %s.\n", domain, username)

	// A build only reaches for these when the image it starts from comes from
	// that registry, so say which images they cover.
	if domain == registry.DefaultDomain {
		fmt.Println("They will be used for images with no registry of their own.")
	}

	return nil
}
