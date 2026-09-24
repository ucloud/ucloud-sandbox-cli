package template

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

func parseTemplateLogsFlags(t *testing.T, args ...string) *logsOperation {
	t.Helper()

	op := &logsOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))

	return op
}

func TestTemplateLogsFlags(t *testing.T) {
	op := parseTemplateLogsFlags(t, "--cursor", "1758693600000", "-l", "50", "--level", "warn")

	require.NotNil(t, op.params.Cursor)
	assert.Equal(t, int64(1758693600000), *op.params.Cursor)
	require.NotNil(t, op.params.Limit)
	assert.Equal(t, int32(50), *op.params.Limit)
	require.NotNil(t, op.params.Level)
	assert.Equal(t, api.LogLevelWarn, *op.params.Level)
}

func TestTemplateLogsValidate(t *testing.T) {
	assert.NoError(t, parseTemplateLogsFlags(t).validate())
	assert.NoError(t, parseTemplateLogsFlags(t, "--level", "debug").validate())
	assert.ErrorContains(t, parseTemplateLogsFlags(t, "--level", "trace").validate(), "invalid --level")
}

// buildLogEntry builds an entry at the given millisecond offset.
func buildLogEntry(ms int64, step, message string) api.BuildLogEntry {
	entry := api.BuildLogEntry{
		Timestamp: time.UnixMilli(ms).UTC(),
		Level:     api.LogLevelInfo,
		Message:   message,
	}
	if step != "" {
		entry.Step = &step
	}

	return entry
}

func TestFormatBuildLogEntry(t *testing.T) {
	stamp := time.Date(2026, time.July, 17, 14, 30, 15, 250_000_000, time.Local)
	step := "RUN"

	assert.Equal(t,
		"2026-07-17 14:30:15.250 [INFO] [RUN] installing packages",
		formatBuildLogEntry(api.BuildLogEntry{
			Timestamp: stamp,
			Level:     api.LogLevelInfo,
			Step:      &step,
			Message:   "installing packages",
		}))
}

func TestFormatBuildLogEntryWithoutAStep(t *testing.T) {
	stamp := time.Date(2026, time.July, 17, 14, 30, 15, 0, time.Local)

	// Step is optional, so a nil one must not panic or print an empty bracket.
	assert.Equal(t,
		"2026-07-17 14:30:15.000 [INFO] starting",
		formatBuildLogEntry(api.BuildLogEntry{
			Timestamp: stamp,
			Level:     api.LogLevelInfo,
			Message:   "starting",
		}))
}

func TestBuildLogSignatureSeparatesTheParts(t *testing.T) {
	// A step of "a" with message "b" must not collide with no step and
	// message "a\x00b"; the separator is what keeps them apart.
	assert.NotEqual(t,
		buildLogSignature(buildLogEntry(0, "a", "b")),
		buildLogSignature(buildLogEntry(0, "", "a\x00b")))
}

// fakeBuildLogsClient serves canned pages and records the cursors it was asked
// for.
type fakeBuildLogsClient struct {
	pages   [][]api.BuildLogEntry
	fetched int

	cursors []int64

	err error
}

func (c *fakeBuildLogsClient) BuildLogs(
	_ context.Context,
	_, _ string,
	params *api.TemplateBuildLogsParams,
) ([]api.BuildLogEntry, error) {
	if c.err != nil {
		return nil, c.err
	}

	cursor := int64(0)
	if params.Cursor != nil {
		cursor = *params.Cursor
	}
	c.cursors = append(c.cursors, cursor)

	if c.fetched >= len(c.pages) {
		return nil, nil
	}

	page := c.pages[c.fetched]
	c.fetched++

	return page, nil
}

func TestPrintBuildLogsWalksEveryPage(t *testing.T) {
	client := &fakeBuildLogsClient{pages: [][]api.BuildLogEntry{
		{buildLogEntry(1000, "FROM", "a"), buildLogEntry(2000, "RUN", "b")},
		{buildLogEntry(2000, "RUN", "b"), buildLogEntry(3000, "RUN", "c")},
	}}

	limit := int32(2)
	params := api.TemplateBuildLogsParams{Limit: &limit}

	var out strings.Builder
	require.NoError(t, printBuildLogs(t.Context(), &out, client, "tpl-1", "build-1", params))

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	require.Len(t, lines, 3, "the replayed entry is printed once")
	assert.True(t, strings.HasSuffix(lines[0], "a"))
	assert.True(t, strings.HasSuffix(lines[1], "b"))
	assert.True(t, strings.HasSuffix(lines[2], "c"))

	assert.Equal(t, []int64{0, 2000, 3000}, client.cursors)
}

func TestPrintBuildLogsStartsAtTheGivenCursor(t *testing.T) {
	client := &fakeBuildLogsClient{pages: [][]api.BuildLogEntry{
		{buildLogEntry(5000, "RUN", "a")},
	}}

	limit := int32(10)
	cursor := int64(4000)
	params := api.TemplateBuildLogsParams{Limit: &limit, Cursor: &cursor}

	var out strings.Builder
	require.NoError(t, printBuildLogs(t.Context(), &out, client, "tpl-1", "build-1", params))

	assert.Equal(t, []int64{4000}, client.cursors)
	assert.Contains(t, out.String(), "a")
}

func TestPrintBuildLogsPropagatesError(t *testing.T) {
	wanted := fmt.Errorf("logs unavailable")

	limit := int32(10)
	params := api.TemplateBuildLogsParams{Limit: &limit}

	var out strings.Builder
	err := printBuildLogs(t.Context(), &out, &fakeBuildLogsClient{err: wanted}, "tpl-1", "build-1", params)
	assert.ErrorIs(t, err, wanted)
}
