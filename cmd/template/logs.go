package template

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-cli/internal/logs"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

const (
	// buildLogsPageSize is the maximum page size accepted by the logs endpoint.
	buildLogsPageSize  = 100
	buildLogTimeFormat = "2006-01-02 15:04:05.000"
)

type logsOperation struct {
	params api.TemplateBuildLogsParams
}

func (o *logsOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "logs <template-id> <build-id>",
		Aliases: []string{"log", "lg"},
		Short:   "Print the logs of a template build",
		Args:    cobra.ExactArgs(2),
	}

	flags.NullableInt64VarP(c.Flags(), &o.params.Cursor, "cursor", "",
		"Millisecond timestamp to start from")
	flags.NullableInt32VarP(c.Flags(), &o.params.Limit, "limit", "l",
		fmt.Sprintf("Maximum number of entries per request (default %d)", buildLogsPageSize))
	flags.NullableStringVarP(c.Flags(), &o.params.Level, "level", "",
		"Minimum log level to print (debug, info, warn, error)")

	return c
}

// validate checks the flag values.
func (o *logsOperation) validate() error {
	if o.params.Level != nil && !o.params.Level.Valid() {
		return fmt.Errorf("invalid --level %q: expected %s, %s, %s or %s",
			*o.params.Level, api.LogLevelDebug, api.LogLevelInfo, api.LogLevelWarn, api.LogLevelError)
	}

	return nil
}

func (o *logsOperation) Run(ctx cmd.OperationContext) error {
	if err := o.validate(); err != nil {
		return err
	}

	params := o.params

	// Walking a build's log means walking it forwards, and a page size the
	// loop can recognise as full.
	direction := api.LogsDirectionForward
	params.Direction = &direction

	if params.Limit == nil || *params.Limit <= 0 {
		limit := int32(buildLogsPageSize)
		params.Limit = &limit
	}

	return printBuildLogs(ctx, os.Stdout, ctx.Client.Templates(), ctx.Args[0], ctx.Args[1], params)
}

// buildLogsClient is the client subset used to read build logs.
type buildLogsClient interface {
	BuildLogs(ctx context.Context, templateID, buildID string, params *api.TemplateBuildLogsParams) ([]api.BuildLogEntry, error)
}

// printBuildLogs walks the build's logs from the cursor to the end.
func printBuildLogs(
	ctx context.Context,
	out io.Writer,
	client buildLogsClient,
	templateID, buildID string,
	params api.TemplateBuildLogsParams,
) error {
	pageSize := int(*params.Limit)

	cursor := logs.NewCursor(
		func(entry api.BuildLogEntry) time.Time { return entry.Timestamp },
		buildLogSignature,
	)
	if params.Cursor != nil {
		cursor.SeekTo(*params.Cursor)
	}

	for {
		page := params
		if ms := cursor.At(); ms > 0 {
			page.Cursor = &ms
		}

		entries, err := client.BuildLogs(ctx, templateID, buildID, &page)
		if err != nil {
			return err
		}

		fresh, more := cursor.Advance(entries, len(entries) >= pageSize)
		for _, entry := range fresh {
			if _, err := fmt.Fprintln(out, formatBuildLogEntry(entry)); err != nil {
				return err
			}
		}

		if !more {
			return nil
		}
	}
}

// buildLogSignature identifies an entry by its content, so the same entry
// returned twice across pages is recognized.
func buildLogSignature(entry api.BuildLogEntry) string {
	return string(entry.Level) + "\x00" + buildLogStep(entry) + "\x00" + entry.Message
}

// buildLogStep is the build step an entry came from, empty when it came from
// none.
func buildLogStep(entry api.BuildLogEntry) string {
	if entry.Step == nil {
		return ""
	}

	return *entry.Step
}

// formatBuildLogEntry renders one entry as "<timestamp> [<level>] [<step>] <message>".
func formatBuildLogEntry(entry api.BuildLogEntry) string {
	var b strings.Builder

	b.WriteString(entry.Timestamp.In(time.Local).Format(buildLogTimeFormat))

	if entry.Level != "" {
		b.WriteString(" [")
		b.WriteString(strings.ToUpper(string(entry.Level)))
		b.WriteString("]")
	}

	if step := buildLogStep(entry); step != "" {
		b.WriteString(" [")
		b.WriteString(step)
		b.WriteString("]")
	}

	if entry.Message != "" {
		b.WriteString(" ")
		b.WriteString(entry.Message)
	}

	return b.String()
}
