package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/auth"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/benchmark"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/sandbox"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/secret"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/snapshot"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/template"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/update"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/volume"
	"github.com/ucloud/ucloud-sandbox-cli/internal/updatecheck"
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

		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			checkForUpdate(cmd)
		},
	}

	c.AddCommand(auth.Command())
	c.AddCommand(benchmark.Command())
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

// noUpdateCheck lists the commands that skip the daily update check: the
// ones about the CLI itself, and the hidden ones a shell runs for completion.
var noUpdateCheck = map[string]bool{
	"update":                        true,
	"version":                       true,
	"help":                          true,
	"completion":                    true,
	cobra.ShellCompRequestCmd:       true,
	cobra.ShellCompNoDescRequestCmd: true,
}

// checkForUpdate tells the user, on the first command of the day, when a
// newer release is out. The notice goes to stderr so output meant for other
// programs on stdout stays as it is.
func checkForUpdate(cmd *cobra.Command) {
	if noUpdateCheck[cmd.Name()] {
		return
	}

	checker, err := updatecheck.New(Version)
	if err != nil {
		return
	}
	checker.Run(cmd.Context(), cmd.ErrOrStderr(), cmd.Root().Name())
}

func main() {
	c := newCommand()

	err := c.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
