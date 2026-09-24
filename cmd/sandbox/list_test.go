package sandbox

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// parseListFlags registers the list flags the way the command does and returns
// the operation they filled in.
func parseListFlags(t *testing.T, args ...string) *listOperation {
	t.Helper()

	op := &listOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))

	return op
}

func TestListFlags(t *testing.T) {
	op := parseListFlags(t,
		"-m", "env=dev",
		"-s", "running", "-s", "paused",
		"-o", "desc",
		"--started-after", "2026-07-17 14:30",
		"-t", "system/base",
		"-p", "2", "-l", "10", "-f", "json",
	)

	require.NotNil(t, op.params.Metadata)
	assert.Equal(t, "env=dev", *op.params.Metadata)

	// Each occurrence of --state adds one value.
	require.NotNil(t, op.params.State)
	assert.Equal(t, []api.SandboxState{"running", "paused"}, *op.params.State)

	require.NotNil(t, op.params.Order)
	assert.Equal(t, api.OrderDirection("desc"), *op.params.Order)

	require.NotNil(t, op.params.StartedAfter)
	assert.Equal(t,
		time.Date(2026, time.July, 17, 14, 30, 0, 0, time.Local),
		*op.params.StartedAfter)

	require.NotNil(t, op.params.Template)
	assert.Equal(t, "system/base", *op.params.Template)

	assert.Equal(t, 2, op.list.Page)
	assert.Equal(t, 10, op.list.Limit)
	assert.Equal(t, list.FormatJSON, op.list.Format)
}

func TestListFiltersStayUnsetByDefault(t *testing.T) {
	op := parseListFlags(t)

	assert.Nil(t, op.params.Metadata)
	assert.Nil(t, op.params.State)
	assert.Nil(t, op.params.Order)
	assert.Nil(t, op.params.StartedAfter)
	assert.Nil(t, op.params.Template)

	assert.Equal(t, 1, op.list.Page)
	assert.Equal(t, 0, op.list.Limit)
	assert.Equal(t, list.FormatPretty, op.list.Format)
}

func TestToListedSandbox(t *testing.T) {
	started := time.Date(2026, time.July, 17, 14, 30, 0, 0, time.Local)
	alias := "base"

	row := toListedSandbox(api.ListedSandbox{
		SandboxID:  "sbx-1",
		State:      api.Running,
		TemplateID: "system/base",
		Alias:      &alias,
		StartedAt:  started,
		EndAt:      started.Add(time.Hour),
		CpuCount:   2,
		MemoryMB:   2048,
	})

	assert.Equal(t, listedSandbox{
		SandboxID:  "sbx-1",
		State:      "running",
		TemplateID: "system/base",
		Alias:      "base",
		StartedAt:  started,
		EndAt:      started.Add(time.Hour),
		CPUCount:   2,
		MemoryMB:   2048,
	}, row)
}

func TestToListedSandboxWithoutAlias(t *testing.T) {
	row := toListedSandbox(api.ListedSandbox{SandboxID: "sbx-1"})

	assert.Empty(t, row.Alias)
}
