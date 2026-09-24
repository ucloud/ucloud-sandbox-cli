package auth

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/auth/registry"
)

// Command returns the root auth command group.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Manage credentials and configuration",
	}
	c.AddCommand(cmd.BuildLocal(&configOperation{}))
	c.AddCommand(cmd.BuildLocal(&loginOperation{}))
	c.AddCommand(cmd.BuildLocal(&logoutOperation{}))
	c.AddCommand(cmd.BuildLocal(&regionOperation{}))
	c.AddCommand(registry.Command())
	return c
}
