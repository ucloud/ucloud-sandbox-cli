package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	fscmd "github.com/ucloud/ucloud-sandbox-cli/cmd/fs"
	sandboxcmd "github.com/ucloud/ucloud-sandbox-cli/cmd/sandbox"
	snapshotcmd "github.com/ucloud/ucloud-sandbox-cli/cmd/snapshot"
	templatecmd "github.com/ucloud/ucloud-sandbox-cli/cmd/template"
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

	c.AddCommand(cmd.NewLoginCmd())
	c.AddCommand(cmd.NewLogoutCmd())
	c.AddCommand(cmd.NewRegionCmd())
	c.AddCommand(cmd.NewConfigCmd())
	c.AddCommand(sandboxcmd.NewSandboxCmd())
	c.AddCommand(fscmd.NewFsCmd())
	c.AddCommand(snapshotcmd.NewSnapshotCmd())
	c.AddCommand(templatecmd.NewTemplateCmd())

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
