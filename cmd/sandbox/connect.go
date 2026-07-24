package sandbox

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
	"golang.org/x/term"
)

func newConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "connect <sandbox-id>",
		Aliases: []string{"conn"},
		Short:   "Connect to a running sandbox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()
			sbx, err := client.ConnectSandbox(ctx, args[0])
			if err != nil {
				return err
			}

			fmt.Printf("Connected to sandbox %s\n", args[0])
			return connectTerminal(ctx, sbx)
		},
	}
}

// connectTerminal starts an interactive PTY session with the sandbox.
func connectTerminal(ctx context.Context, sbx *sdk.Sandbox) error {
	fd := int(os.Stdin.Fd())
	cols, rows, err := term.GetSize(fd)
	if err != nil {
		cols, rows = 80, 24
	}

	handle, err := sbx.Pty.Create(ctx, sdk.PtySize{Cols: cols, Rows: rows})
	if err != nil {
		return err
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, oldState)

	// Forward PTY output to stdout.
	go func() {
		for ev := range handle.Events() {
			os.Stdout.Write(ev.Data)
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

	handle.Wait()
	return nil
}
