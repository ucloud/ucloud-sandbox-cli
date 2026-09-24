package snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

func TestCommandsAreRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Command().Commands() {
		names[c.Name()] = true
	}

	for _, name := range []string{"create", "delete", "list"} {
		assert.True(t, names[name], "snapshot %s is missing", name)
	}
}

func TestCreateFlags(t *testing.T) {
	op := &createOperation{}
	c := op.Command()

	require.NoError(t, c.ParseFlags(nil))
	assert.Nil(t, op.req.Name, "an unnamed snapshot gets one of the platform's making")

	require.NoError(t, c.ParseFlags([]string{"-n", "nightly"}))
	require.NotNil(t, op.req.Name)
	assert.Equal(t, "nightly", *op.req.Name)
}

func TestCreateArgCount(t *testing.T) {
	c := (&createOperation{}).Command()

	assert.Error(t, c.Args(c, nil))
	assert.NoError(t, c.Args(c, []string{"sbx-1"}))
	assert.Error(t, c.Args(c, []string{"sbx-1", "extra"}))
}

func TestDeleteNeedsAtLeastOneSnapshot(t *testing.T) {
	c := (&deleteOperation{}).Command()

	assert.Error(t, c.Args(c, nil))
	assert.NoError(t, c.Args(c, []string{"snap-1"}))
	assert.NoError(t, c.Args(c, []string{"snap-1", "snap-2"}))
}

func TestListFlags(t *testing.T) {
	op := &listOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"-s", "sbx-1", "-n", "nightly", "-f", "json"}))

	require.NotNil(t, op.params.SandboxID)
	assert.Equal(t, "sbx-1", *op.params.SandboxID)
	require.NotNil(t, op.params.Name)
	assert.Equal(t, "nightly", *op.params.Name)
	assert.Equal(t, list.FormatJSON, op.list.Format)
}

func TestListFiltersStayUnsetByDefault(t *testing.T) {
	op := &listOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(nil))

	assert.Nil(t, op.params.SandboxID)
	assert.Nil(t, op.params.Name)
}

func TestToListedSnapshot(t *testing.T) {
	row := toListedSnapshot(api.SnapshotInfo{
		SnapshotID: "team/nightly:default",
		Names:      []string{"team/nightly:v2", "nightly"},
	})

	assert.Equal(t, listedSnapshot{
		SnapshotID: "team/nightly:default",
		Names:      "team/nightly:v2, nightly",
	}, row)
}
