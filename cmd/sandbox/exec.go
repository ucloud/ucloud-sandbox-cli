package sandbox

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox/commands"
)

type execOperation struct {
	req api.ConnectSandbox

	user            string
	commandsOptions commands.Options
}

func (o *execOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "exec <sandbox-id> <command>",
		Aliases: []string{"ex"},
		Short:   "Execute a command in a sandbox",
		Args:    cobra.MinimumNArgs(2),
	}

	// Everything after the sandbox ID belongs to the remote command, so the
	// CLI's own flags have to come before it:
	//
	//	ucloud-sandbox-cli sandbox exec -u root <sandbox-id> ls -la
	c.Flags().SetInterspersed(false)

	connectSandboxVarP(c.Flags(), &o.req)
	sandboxUserVarP(c.Flags(), &o.user)
	commandsOptionsVarP(c.Flags(), &o.commandsOptions)

	c.Flags().IntVarP(&o.commandsOptions.TimeoutSeconds, "command-timeout", "", 0,
		"Seconds after which the command is killed (0 uses the default, a negative value disables it)")
	c.Flags().BoolVarP(&o.commandsOptions.Stdin, "stdin", "", false, "Keep the command's stdin open")

	return c
}

func (o *execOperation) Run(ctx cmd.OperationContext) error {
	command, err := buildCommand(ctx.Args[1:])
	if err != nil {
		return err
	}

	// Connecting resumes a paused sandbox, so the command can be run either
	// way.
	sbx, err := ctx.Client.Sandboxes().Connect(ctx, ctx.Args[0], o.req)
	if err != nil {
		return err
	}

	envd, err := ctx.Client.Sandboxes().Envd(sbx, o.user)
	if err != nil {
		return fmt.Errorf("failed to connect to sandbox envd: %w", err)
	}

	// The command's output is this process' output, as it arrives.
	opts := o.commandsOptions
	opts.OnStdout = func(s string) { fmt.Fprint(os.Stdout, s) }
	opts.OnStderr = func(s string) { fmt.Fprint(os.Stderr, s) }

	// Run reports a non-zero exit as an error; the output it carries has
	// already been streamed by the callbacks above.
	_, err = envd.Commands().Run(ctx, command, opts)

	return err
}

// shellSafe matches an argument that needs no quoting for the shell envd runs
// the command through.
var shellSafe = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// shellQuote quotes one argument so the remote shell reads it as a single
// word.
func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}

	if shellSafe.MatchString(arg) {
		return arg
	}

	return "'" + strings.ReplaceAll(arg, "'", `'"'"'`) + "'"
}

// buildCommand joins the command parts into the one line the remote shell
// runs. A single part is passed through untouched, so a quoted command line
// keeps its pipes and redirections; several parts are quoted one by one, so an
// argument carrying spaces stays one argument.
func buildCommand(parts []string) (string, error) {
	// "--" only separates the CLI's own flags from the command.
	if len(parts) > 0 && parts[0] == "--" {
		parts = parts[1:]
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("missing command to execute")
	}

	if len(parts) == 1 {
		return parts[0], nil
	}

	quoted := make([]string, len(parts))
	for i, part := range parts {
		quoted[i] = shellQuote(part)
	}

	return strings.Join(quoted, " "), nil
}
