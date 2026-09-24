package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/auth"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/sandbox"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/secret"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/snapshot"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/template"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/update"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/volume"
)

var (
	Version string
	Commit  string
)

func newCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "ucloud-sandbox-cli",
		Short: "Commands to manage UCloud sandbox, visit https://astraflow.ucloud.cn/docs/agent-sandbox/product/cli for more help",

		SilenceErrors: true,
		SilenceUsage:  true,

		Version: Version,
	}

	c.AddCommand(auth.Command())
	c.AddCommand(sandbox.Command())
	c.AddCommand(secret.Command())
	c.AddCommand(snapshot.Command())
	c.AddCommand(template.Command())
	c.AddCommand(update.Command(Version))
	c.AddCommand(volume.Command())

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version and commit",

		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ucloud-sandbox-cli %s, commit: %s\n", Version, Commit)
		},
	}
	c.AddCommand(versionCmd)

	return c
}

func main() {
	c := newCommand()

	err := c.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
