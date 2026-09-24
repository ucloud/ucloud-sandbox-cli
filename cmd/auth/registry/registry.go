package registry

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/registry"
)

// Command returns the registry command group, a subcommand of auth.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "registry",
		Aliases: []string{"reg"},
		Short:   "Manage container registry credentials",
	}
	c.AddCommand(cmd.BuildLocal(&loginOperation{}))
	c.AddCommand(cmd.BuildLocal(&logoutOperation{}))
	return c
}

// domainOf reads the registry domain off the command line, falling back to the
// registry a bare image name refers to.
func domainOf(args []string) (string, error) {
	if len(args) == 0 {
		return registry.DefaultDomain, nil
	}

	return registry.NormalizeDomain(args[0])
}
