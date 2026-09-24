//go:build windows

package sandbox

import (
	"context"
	"time"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox/pty"
	"golang.org/x/term"
)

func watchTerminalResize(ctx context.Context, fd int, handle *pty.Handle, cols, rows int) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				newCols, newRows, err := term.GetSize(fd)
				if err != nil || (newCols == cols && newRows == rows) {
					continue
				}
				if err := handle.Resize(ctx, pty.Size{Cols: newCols, Rows: newRows}); err == nil {
					cols, rows = newCols, newRows
				}
			}
		}
	}()

	return func() { close(done) }
}
