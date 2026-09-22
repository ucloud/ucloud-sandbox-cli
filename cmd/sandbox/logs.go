package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
)

const (
	// sandboxLogsPageSize is the maximum page size accepted by the logs endpoint.
	sandboxLogsPageSize   = 1000
	sandboxLogTimeFormat  = "2006-01-02 15:04:05.000"
	sandboxLogsPollPeriod = time.Second
)

func newLogsCmd() *cobra.Command {
	var level, search string
	var follow bool

	cmd := &cobra.Command{
		Use:     "logs <sandbox-id>",
		Aliases: []string{"log", "lg"},
		Short:   "Print the logs of a sandbox",
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

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			if follow {
				// Ctrl+C ends the stream without failing the command.
				var stop context.CancelFunc
				ctx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
				defer stop()
			}

			return printSandboxLogs(ctx, client, args[0], level, search, follow)
		},
	}

	cmd.Flags().StringVar(&level, "level", "", "Minimum log level (debug, info, warn, error)")
	cmd.Flags().StringVar(&search, "search", "", "Only print entries whose message contains this substring")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Keep streaming logs until the sandbox stops")
	return cmd
}

func printSandboxLogs(ctx context.Context, client *client.Client, sandboxID, level, search string, follow bool) error {
	cursor := newSandboxLogsCursor()
	// draining is the final pass after the sandbox stopped, so that entries
	// written just before the end are still reported.
	draining := false

	for {
		opts := sandbox.LogsV2Options{
			Limit:     sandboxLogsPageSize,
			Direction: sandbox.LogsDirectionForward,
		}
		if cursor.ms > 0 {
			opts.CursorMs = new(cursor.ms)
		}
		if level != "" {
			opts.Level = level
		}
		if search != "" {
			opts.Search = search
		}

		page, err := client.Sandboxes().LogsV2(ctx, sandboxID, opts)
		if err != nil {
			return err
		}

		fresh, more := cursor.advance(page, len(page) >= sandboxLogsPageSize)
		for _, entry := range fresh {
			fmt.Println(formatSandboxLogEntry(entry))
		}
		if more {
			continue
		}
		if !follow || draining {
			return nil
		}

		running, err := isSandboxRunning(ctx, client, sandboxID)
		if err != nil {
			return err
		}
		draining = !running

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(sandboxLogsPollPeriod):
		}
	}
}

// isSandboxRunning reports whether the sandbox still produces logs. A sandbox
// that no longer exists counts as stopped rather than as an error.
func isSandboxRunning(ctx context.Context, client *client.Client, sandboxID string) (bool, error) {
	info, err := client.Sandboxes().Get(ctx, sandboxID)
	if err != nil {
		var notFound *errdefs.NotFoundError
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}
	return strings.EqualFold(string(info.State), "running"), nil
}

// sandboxLogsCursor tracks the forward pagination position of sandbox logs. The
// endpoint takes a millisecond timestamp as its cursor, so entries sharing the
// millisecond of the previous page's last entry are returned again and have to
// be filtered out.
type sandboxLogsCursor struct {
	ms   int64
	seen map[string]struct{}
}

func newSandboxLogsCursor() *sandboxLogsCursor {
	return &sandboxLogsCursor{seen: map[string]struct{}{}}
}

// advance returns the entries of page that have not been reported yet and
// whether another page should be requested right away. pageFull tells whether
// the page reached the requested limit, meaning more entries may be waiting.
func (c *sandboxLogsCursor) advance(page []sandbox.LogEntry, pageFull bool) ([]sandbox.LogEntry, bool) {
	if len(page) == 0 {
		return nil, false
	}

	fresh := make([]sandbox.LogEntry, 0, len(page))
	for _, entry := range page {
		ms := entry.Timestamp.UnixMilli()
		if ms < c.ms {
			continue
		}
		if ms == c.ms {
			if _, ok := c.seen[sandboxLogSignature(entry)]; ok {
				continue
			}
		}
		fresh = append(fresh, entry)
	}

	lastMs := page[len(page)-1].Timestamp.UnixMilli()
	if pageFull && lastMs == c.ms && len(fresh) == 0 {
		// A full page holds nothing but already reported entries of the cursor
		// millisecond. Step over it, otherwise the same page repeats forever.
		c.ms++
		c.seen = map[string]struct{}{}
		return fresh, true
	}

	if lastMs != c.ms {
		c.ms = lastMs
		c.seen = map[string]struct{}{}
	}
	for _, entry := range page {
		if entry.Timestamp.UnixMilli() == c.ms {
			c.seen[sandboxLogSignature(entry)] = struct{}{}
		}
	}
	return fresh, pageFull
}

func sandboxLogSignature(entry sandbox.LogEntry) string {
	var b strings.Builder
	b.WriteString(entry.Level)
	b.WriteString("\x00")
	b.WriteString(entry.Message)
	for _, key := range sortedFieldKeys(entry.Fields) {
		b.WriteString("\x00")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(entry.Fields[key])
	}
	return b.String()
}

// formatSandboxLogEntry renders one entry as "<timestamp> [<level>] <message> <key=value...>".
func formatSandboxLogEntry(entry sandbox.LogEntry) string {
	var b strings.Builder
	b.WriteString(entry.Timestamp.In(time.Local).Format(sandboxLogTimeFormat))
	if entry.Level != "" {
		b.WriteString(" [")
		b.WriteString(strings.ToUpper(entry.Level))
		b.WriteString("]")
	}
	if entry.Message != "" {
		b.WriteString(" ")
		b.WriteString(entry.Message)
	}
	for _, key := range sortedFieldKeys(entry.Fields) {
		b.WriteString(" ")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(entry.Fields[key])
	}
	return b.String()
}

func sortedFieldKeys(fields map[string]string) []string {
	if len(fields) == 0 {
		return nil
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
