package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-cli/internal/logs"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

const (
	// sandboxLogsPageSize is the maximum page size accepted by the logs endpoint.
	sandboxLogsPageSize   = 1000
	sandboxLogTimeFormat  = "2006-01-02 15:04:05.000"
	sandboxLogsPollPeriod = time.Second
)

type logsOperation struct {
	params api.SandboxLogsParamsV2

	follow bool
}

func (o *logsOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "logs <sandbox-id>",
		Aliases: []string{"log", "lg"},
		Short:   "Print the logs of a sandbox",
		Args:    cobra.ExactArgs(1),
	}

	flags.NullableInt64VarP(c.Flags(), &o.params.Cursor, "cursor", "",
		"Millisecond timestamp to start from")
	flags.NullableStringVarP(c.Flags(), &o.params.Direction, "direction", "d",
		fmt.Sprintf("Direction to walk the log in (forward, backward) (default %s)", api.LogsDirectionForward))
	flags.NullableInt32VarP(c.Flags(), &o.params.Limit, "limit", "l",
		fmt.Sprintf("Maximum number of entries per request (default %d)", sandboxLogsPageSize))
	flags.NullableStringVarP(c.Flags(), &o.params.Level, "level", "",
		"Minimum log level to print (debug, info, warn, error)")
	flags.NullableStringVarP(c.Flags(), &o.params.Search, "search", "",
		"Only print entries whose message contains this substring")

	c.Flags().BoolVarP(&o.follow, "follow", "f", false, "Keep streaming logs until the sandbox stops")

	return c
}

// validate checks the flag values against each other.
func (o *logsOperation) validate() error {
	if o.params.Direction != nil && !o.params.Direction.Valid() {
		return fmt.Errorf("invalid --direction %q: expected %s or %s",
			*o.params.Direction, api.LogsDirectionForward, api.LogsDirectionBackward)
	}

	if o.params.Level != nil && !o.params.Level.Valid() {
		return fmt.Errorf("invalid --level %q: expected %s, %s, %s or %s",
			*o.params.Level, api.LogLevelDebug, api.LogLevelInfo, api.LogLevelWarn, api.LogLevelError)
	}

	// Following means walking the cursor towards newer entries, which only the
	// forward direction does.
	if o.follow && o.params.Direction != nil && *o.params.Direction == api.LogsDirectionBackward {
		return fmt.Errorf("--follow needs --direction %s", api.LogsDirectionForward)
	}

	return nil
}

// defaulted returns the request params with the values the streaming loop
// relies on filled in.
func (o *logsOperation) defaulted() api.SandboxLogsParamsV2 {
	params := o.params

	if params.Direction == nil {
		direction := api.LogsDirectionForward
		params.Direction = &direction
	}

	if params.Limit == nil || *params.Limit <= 0 {
		limit := int32(sandboxLogsPageSize)
		params.Limit = &limit
	}

	return params
}

func (o *logsOperation) Run(ctx cmd.OperationContext) error {
	if err := o.validate(); err != nil {
		return err
	}

	params := o.defaulted()
	client := ctx.Client.Sandboxes()

	// Walking backwards starts at the newest entry, so there is no cursor to
	// follow and one request is the whole answer.
	if *params.Direction == api.LogsDirectionBackward {
		entries, err := client.LogsV2(ctx, ctx.Args[0], &params)
		if err != nil {
			return err
		}

		return printSandboxLogs(os.Stdout, entries)
	}

	var streamCtx context.Context = ctx
	if o.follow {
		// Ctrl+C ends the stream without failing the command.
		var stop context.CancelFunc
		streamCtx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()
	}

	return streamSandboxLogs(streamCtx, os.Stdout, client, ctx.Args[0], params, o.follow)
}

// printSandboxLogs writes the entries one formatted line each.
func printSandboxLogs(out io.Writer, entries []api.SandboxLogEntry) error {
	for _, entry := range entries {
		if _, err := fmt.Fprintln(out, formatSandboxLogEntry(entry)); err != nil {
			return err
		}
	}

	return nil
}

// logsClient is the client subset used to read sandbox logs.
type logsClient interface {
	LogsV2(ctx context.Context, sandboxID string, params *api.SandboxLogsParamsV2) ([]api.SandboxLogEntry, error)
	Get(ctx context.Context, sandboxID string) (*api.SandboxDetail, error)
}

// streamSandboxLogs prints the sandbox's logs from the cursor onwards. Without
// follow it returns once the listing is exhausted; with it, it keeps polling
// until the sandbox stops or ctx is done.
func streamSandboxLogs(
	ctx context.Context,
	out io.Writer,
	client logsClient,
	sandboxID string,
	params api.SandboxLogsParamsV2,
	follow bool,
) error {
	pageSize := int(*params.Limit)

	cursor := newSandboxLogsCursor()
	if params.Cursor != nil {
		cursor.SeekTo(*params.Cursor)
	}

	// draining is the final pass after the sandbox stopped, so that entries
	// written just before the end are still reported.
	draining := false

	for {
		page := params
		if ms := cursor.At(); ms > 0 {
			page.Cursor = &ms
		}

		entries, err := client.LogsV2(ctx, sandboxID, &page)
		if err != nil {
			return err
		}

		fresh, more := cursor.Advance(entries, len(entries) >= pageSize)
		if err := printSandboxLogs(out, fresh); err != nil {
			return err
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
func isSandboxRunning(ctx context.Context, client logsClient, sandboxID string) (bool, error) {
	detail, err := client.Get(ctx, sandboxID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return detail.State == api.Running, nil
}

// newSandboxLogsCursor returns the pagination cursor for a sandbox's logs.
func newSandboxLogsCursor() *logs.Cursor[api.SandboxLogEntry] {
	return logs.NewCursor(
		func(entry api.SandboxLogEntry) time.Time { return entry.Timestamp },
		sandboxLogSignature,
	)
}

// sandboxLogSignature identifies an entry by its content, so the same entry
// returned twice across pages is recognized.
func sandboxLogSignature(entry api.SandboxLogEntry) string {
	var b strings.Builder

	b.WriteString(string(entry.Level))
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
func formatSandboxLogEntry(entry api.SandboxLogEntry) string {
	var b strings.Builder

	b.WriteString(entry.Timestamp.In(time.Local).Format(sandboxLogTimeFormat))

	if entry.Level != "" {
		b.WriteString(" [")
		b.WriteString(strings.ToUpper(string(entry.Level)))
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
