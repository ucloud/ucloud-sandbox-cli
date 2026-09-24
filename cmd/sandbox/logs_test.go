package sandbox

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// parseLogsFlags registers the logs flags the way the command does and returns
// the operation they filled in.
func parseLogsFlags(t *testing.T, args ...string) *logsOperation {
	t.Helper()

	op := &logsOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))

	return op
}

func TestLogsFlags(t *testing.T) {
	op := parseLogsFlags(t,
		"--cursor", "1758693600000",
		"-d", "backward",
		"-l", "50",
		"--level", "warn",
		"--search", "panic",
		"sbx-1",
	)

	require.NotNil(t, op.params.Cursor)
	assert.Equal(t, int64(1758693600000), *op.params.Cursor)
	require.NotNil(t, op.params.Direction)
	assert.Equal(t, api.LogsDirectionBackward, *op.params.Direction)
	require.NotNil(t, op.params.Limit)
	assert.Equal(t, int32(50), *op.params.Limit)
	require.NotNil(t, op.params.Level)
	assert.Equal(t, api.LogLevelWarn, *op.params.Level)
	require.NotNil(t, op.params.Search)
	assert.Equal(t, "panic", *op.params.Search)
	assert.False(t, op.follow)
}

func TestLogsValidate(t *testing.T) {
	tests := []struct {
		name    string
		flags   []string
		wantErr string
	}{
		{name: "no flags"},
		{name: "known level and direction", flags: []string{"--level", "debug", "-d", "forward"}},
		{name: "unknown direction", flags: []string{"-d", "sideways"}, wantErr: "invalid --direction"},
		{name: "unknown level", flags: []string{"--level", "trace"}, wantErr: "invalid --level"},
		{
			name:    "follow backwards",
			flags:   []string{"-f", "-d", "backward"},
			wantErr: "--follow needs --direction forward",
		},
		{name: "follow forwards", flags: []string{"-f", "-d", "forward"}},
		{name: "follow without a direction", flags: []string{"-f"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := parseLogsFlags(t, tt.flags...)

			err := op.validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLogsDefaulted(t *testing.T) {
	params := parseLogsFlags(t).defaulted()

	require.NotNil(t, params.Direction)
	assert.Equal(t, api.LogsDirectionForward, *params.Direction)
	require.NotNil(t, params.Limit)
	assert.Equal(t, int32(sandboxLogsPageSize), *params.Limit)

	// A limit that would make every page count as full is replaced, otherwise
	// the paging loop would never end.
	params = parseLogsFlags(t, "-l", "0").defaulted()
	require.NotNil(t, params.Limit)
	assert.Equal(t, int32(sandboxLogsPageSize), *params.Limit)

	params = parseLogsFlags(t, "-l", "5", "-d", "backward").defaulted()
	require.NotNil(t, params.Limit)
	assert.Equal(t, int32(5), *params.Limit)
	assert.Equal(t, api.LogsDirectionBackward, *params.Direction)
}

// logEntry builds an entry at the given millisecond offset from a fixed base.
func logEntry(ms int64, level api.LogLevel, message string) api.SandboxLogEntry {
	return api.SandboxLogEntry{
		Timestamp: time.UnixMilli(ms).UTC(),
		Level:     level,
		Message:   message,
	}
}

func TestFormatSandboxLogEntry(t *testing.T) {
	stamp := time.Date(2026, time.July, 17, 14, 30, 15, 250_000_000, time.Local)

	entry := api.SandboxLogEntry{
		Timestamp: stamp,
		Level:     api.LogLevelWarn,
		Message:   "disk almost full",
		Fields:    map[string]string{"used": "90%", "mount": "/"},
	}

	// Fields are sorted so the same entry always renders the same way.
	assert.Equal(t,
		"2026-07-17 14:30:15.250 [WARN] disk almost full mount=/ used=90%",
		formatSandboxLogEntry(entry))
}

func TestFormatSandboxLogEntryOmitsEmptyParts(t *testing.T) {
	stamp := time.Date(2026, time.July, 17, 14, 30, 15, 0, time.Local)

	assert.Equal(t, "2026-07-17 14:30:15.000",
		formatSandboxLogEntry(api.SandboxLogEntry{Timestamp: stamp}))
}

// fakeLogsClient serves canned pages and records the cursors it was asked for.
type fakeLogsClient struct {
	pages   [][]api.SandboxLogEntry
	fetched int

	cursors []int64

	state    api.SandboxState
	getErr   error
	getCalls int
}

func (c *fakeLogsClient) LogsV2(
	_ context.Context,
	_ string,
	params *api.SandboxLogsParamsV2,
) ([]api.SandboxLogEntry, error) {
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

func (c *fakeLogsClient) Get(_ context.Context, _ string) (*api.SandboxDetail, error) {
	c.getCalls++

	if c.getErr != nil {
		return nil, c.getErr
	}

	return &api.SandboxDetail{State: c.state}, nil
}

func TestStreamSandboxLogsWalksEveryPage(t *testing.T) {
	client := &fakeLogsClient{pages: [][]api.SandboxLogEntry{
		{logEntry(1000, api.LogLevelInfo, "a"), logEntry(2000, api.LogLevelInfo, "b")},
		{logEntry(2000, api.LogLevelInfo, "b"), logEntry(3000, api.LogLevelInfo, "c")},
	}}

	limit := int32(2)
	params := api.SandboxLogsParamsV2{Limit: &limit}

	var out strings.Builder
	require.NoError(t, streamSandboxLogs(t.Context(), &out, client, "sbx-1", params, false))

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	require.Len(t, lines, 3, "the replayed entry is printed once")
	assert.True(t, strings.HasSuffix(lines[0], "a"))
	assert.True(t, strings.HasSuffix(lines[1], "b"))
	assert.True(t, strings.HasSuffix(lines[2], "c"))

	// The first request starts where the caller asked, the later ones follow
	// the cursor.
	assert.Equal(t, []int64{0, 2000, 3000}, client.cursors)
	assert.Zero(t, client.getCalls, "the sandbox's state only matters when following")
}

func TestStreamSandboxLogsStartsAtTheGivenCursor(t *testing.T) {
	client := &fakeLogsClient{pages: [][]api.SandboxLogEntry{
		{logEntry(5000, api.LogLevelInfo, "a")},
	}}

	limit := int32(10)
	cursor := int64(4000)
	params := api.SandboxLogsParamsV2{Limit: &limit, Cursor: &cursor}

	var out strings.Builder
	require.NoError(t, streamSandboxLogs(t.Context(), &out, client, "sbx-1", params, false))

	assert.Equal(t, []int64{4000}, client.cursors)
	assert.Contains(t, out.String(), "a")
}

func TestStreamSandboxLogsPropagatesError(t *testing.T) {
	wanted := fmt.Errorf("logs unavailable")

	client := &erroringLogsClient{err: wanted}

	limit := int32(10)
	params := api.SandboxLogsParamsV2{Limit: &limit}

	var out strings.Builder
	err := streamSandboxLogs(t.Context(), &out, client, "sbx-1", params, false)
	assert.ErrorIs(t, err, wanted)
}

type erroringLogsClient struct{ err error }

func (c *erroringLogsClient) LogsV2(
	_ context.Context,
	_ string,
	_ *api.SandboxLogsParamsV2,
) ([]api.SandboxLogEntry, error) {
	return nil, c.err
}

func (c *erroringLogsClient) Get(_ context.Context, _ string) (*api.SandboxDetail, error) {
	return nil, c.err
}

func TestIsSandboxRunning(t *testing.T) {
	running := &fakeLogsClient{state: api.Running}
	got, err := isSandboxRunning(t.Context(), running, "sbx-1")
	require.NoError(t, err)
	assert.True(t, got)

	paused := &fakeLogsClient{state: api.Paused}
	got, err = isSandboxRunning(t.Context(), paused, "sbx-1")
	require.NoError(t, err)
	assert.False(t, got)
}

func TestIsSandboxRunningTreatsAMissingSandboxAsStopped(t *testing.T) {
	gone := &fakeLogsClient{getErr: &errdefs.NotFoundError{}}

	got, err := isSandboxRunning(t.Context(), gone, "sbx-1")
	require.NoError(t, err)
	assert.False(t, got)
}

func TestIsSandboxRunningPropagatesOtherErrors(t *testing.T) {
	wanted := fmt.Errorf("control plane unreachable")

	got, err := isSandboxRunning(t.Context(), &fakeLogsClient{getErr: wanted}, "sbx-1")
	assert.ErrorIs(t, err, wanted)
	assert.False(t, got)
}
