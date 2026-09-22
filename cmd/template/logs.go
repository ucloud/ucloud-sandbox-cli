package template

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/template"
)

const (
	// buildLogsPageSize is the maximum page size accepted by the logs endpoint.
	buildLogsPageSize  = 100
	buildLogTimeFormat = "2006-01-02 15:04:05.000"
)

func newLogsCmd() *cobra.Command {
	var level string

	cmd := &cobra.Command{
		Use:     "logs <template-id> <build-id>",
		Aliases: []string{"log"},
		Short:   "Print the build logs of a template build",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("template and build are required")
			}
			if len(args) > 2 {
				return fmt.Errorf("only one build can be specified")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			return printBuildLogs(cmd.Context(), client, args[0], args[1], level)
		},
	}

	cmd.Flags().StringVar(&level, "level", "", "Minimum log level (debug, info, warn, error)")
	return cmd
}

func printBuildLogs(ctx context.Context, client *client.Client, templateID, buildID, level string) error {
	cursor := newBuildLogsCursor()
	for {
		opts := template.BuildLogsOptions{
			Limit:     buildLogsPageSize,
			Direction: template.LogsDirectionForward,
		}
		if cursor.ms > 0 {
			opts.CursorMs = new(cursor.ms)
		}
		if level != "" {
			opts.Level = template.LogLevel(level)
		}

		page, err := client.Templates().BuildLogs(ctx, templateID, buildID, opts)
		if err != nil {
			return err
		}

		fresh, more := cursor.advance(page, len(page) >= buildLogsPageSize)
		for _, entry := range fresh {
			fmt.Println(formatBuildLogEntry(entry))
		}
		if !more {
			return nil
		}
	}
}

// buildLogsCursor tracks the forward pagination position of build logs. The
// endpoint takes a millisecond timestamp as its cursor, so entries sharing the
// millisecond of the previous page's last entry are returned again and have to
// be filtered out.
type buildLogsCursor struct {
	ms   int64
	seen map[string]struct{}
}

func newBuildLogsCursor() *buildLogsCursor {
	return &buildLogsCursor{seen: map[string]struct{}{}}
}

// advance returns the entries of page that have not been reported yet and
// whether another page should be requested. pageFull tells whether the page
// reached the requested limit, meaning more entries may be waiting.
func (c *buildLogsCursor) advance(page []template.LogEntry, pageFull bool) ([]template.LogEntry, bool) {
	if len(page) == 0 {
		return nil, false
	}

	fresh := make([]template.LogEntry, 0, len(page))
	for _, entry := range page {
		ms := entry.Timestamp.UnixMilli()
		if ms < c.ms {
			continue
		}
		if ms == c.ms {
			if _, ok := c.seen[buildLogSignature(entry)]; ok {
				continue
			}
		}
		fresh = append(fresh, entry)
	}

	if !pageFull {
		return fresh, false
	}

	lastMs := page[len(page)-1].Timestamp.UnixMilli()
	if lastMs == c.ms && len(fresh) == 0 {
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
			c.seen[buildLogSignature(entry)] = struct{}{}
		}
	}
	return fresh, true
}

func buildLogSignature(entry template.LogEntry) string {
	return string(entry.Level) + "\x00" + entry.Step + "\x00" + entry.Message
}

// formatBuildLogEntry renders one entry as "<timestamp> [<level>] [<step>] <message>".
func formatBuildLogEntry(entry template.LogEntry) string {
	var b strings.Builder
	b.WriteString(entry.Timestamp.In(time.Local).Format(buildLogTimeFormat))
	if entry.Level != "" {
		b.WriteString(" [")
		b.WriteString(strings.ToUpper(string(entry.Level)))
		b.WriteString("]")
	}
	if entry.Step != "" {
		b.WriteString(" [")
		b.WriteString(entry.Step)
		b.WriteString("]")
	}
	if entry.Message != "" {
		b.WriteString(" ")
		b.WriteString(entry.Message)
	}
	return b.String()
}
