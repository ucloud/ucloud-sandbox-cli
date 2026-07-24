//go:build darwin || linux

package sandbox

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
	"golang.org/x/term"
)

func watchTerminalResize(ctx context.Context, fd int, handle *sdk.PtyHandle, _, _ int) func() {
	sigCh := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(sigCh, syscall.SIGWINCH)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-sigCh:
				if cols, rows, err := term.GetSize(fd); err == nil {
					_ = handle.Resize(ctx, sdk.PtySize{Cols: cols, Rows: rows})
				}
			}
		}
	}()

	return func() {
		signal.Stop(sigCh)
		close(done)
	}
}
