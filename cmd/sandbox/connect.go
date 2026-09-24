package sandbox

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox/commands"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox/pty"
	"golang.org/x/term"
)

// Terminal size used when stdin reports none.
const (
	defaultCols = 80
	defaultRows = 24
)

type connectOperation struct {
	req api.ConnectSandbox

	detached bool

	user            string
	commandsOptions commands.Options
}

func (o *connectOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "connect <sandbox-id>",
		Aliases: []string{"conn"},
		Short:   "Connect a terminal to a sandbox",
		Args:    cobra.ExactArgs(1),
	}

	connectSandboxVarP(c.Flags(), &o.req)

	c.Flags().BoolVarP(&o.detached, "detached", "", false, "Show the connected sandbox instead of connecting a terminal to it")

	sandboxUserVarP(c.Flags(), &o.user)
	commandsOptionsVarP(c.Flags(), &o.commandsOptions)

	return c
}

// The registration helpers below are shared by every command that connects to
// a sandbox and runs something in it — connect, create, and later exec.

// connectSandboxVarP registers the flags of a connect request.
func connectSandboxVarP(fs *pflag.FlagSet, req *api.ConnectSandbox) {
	fs.Int32VarP(&req.Timeout, "timeout", "", 0, "Seconds from now after which the sandbox expires")
	flags.NullableBoolVarP(fs, &req.Memory, "memory", "", "Restore the memory of a paused sandbox")
}

// sandboxUserVarP registers the flag picking the user a session runs as. An
// empty user leaves the choice to the sandbox.
func sandboxUserVarP(fs *pflag.FlagSet, user *string) {
	fs.StringVarP(user, "user", "u", "", "User to start the session as")
}

// commandsOptionsVarP registers the flags shaping a command or terminal
// session inside a sandbox.
func commandsOptionsVarP(fs *pflag.FlagSet, opts *commands.Options) {
	fs.StringVarP(&opts.Cwd, "cwd", "c", "", "Working directory of the session")
	flags.StringMapVarP(fs, &opts.EnvVars, "env", "e", "Environment variable of the session")
}

func (o *connectOperation) Run(ctx cmd.OperationContext) error {
	// Connecting resumes a paused sandbox, so the terminal can be opened
	// either way.
	sbx, err := ctx.Client.Sandboxes().Connect(ctx, ctx.Args[0], o.req)
	if err != nil {
		return err
	}

	fmt.Printf("Sandbox connected: %s\n", sbx.SandboxID)
	if o.detached {
		return cmd.ShowJSON(sbx)
	}

	fmt.Printf("Terminal connecting to sandbox %s\n", sbx.SandboxID)

	if err := connectTerminal(ctx, sbx, o.user, o.commandsOptions); err != nil {
		return err
	}

	fmt.Printf("Closed the terminal of sandbox %s\n", sbx.SandboxID)

	return nil
}

func connectTerminal(ctx cmd.OperationContext, sbx *api.Sandbox, user string, opts commands.Options) error {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return fmt.Errorf("connect needs a terminal on stdin")
	}

	cols, rows, err := term.GetSize(fd)
	if err != nil {
		cols, rows = defaultCols, defaultRows
	}

	envd, err := ctx.Client.Sandboxes().Envd(sbx, user)
	if err != nil {
		return fmt.Errorf("failed to connect to sandbox envd: %w", err)
	}

	ptySize := pty.Size{
		Rows: rows,
		Cols: cols,
	}
	handle, err := envd.Pty().Create(ctx, ptySize, opts)
	if err != nil {
		return fmt.Errorf("failed to create pty: %w", err)
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, oldState)

	// Forward PTY output to stdout.
	go func() {
		for data := range handle.Output() {
			os.Stdout.Write(data)
		}
	}()

	// Forward stdin to PTY.
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			if err := handle.SendStdin(ctx, buf[:n]); err != nil {
				return
			}
		}
	}()

	stopResize := watchTerminalResize(ctx, fd, handle, cols, rows)
	defer stopResize()

	// The shell's own exit code is not the CLI's: a session ended by "exit 1"
	// is still a session that ran.
	if _, err := handle.Wait(ctx); err != nil {
		return err
	}

	return nil
}
