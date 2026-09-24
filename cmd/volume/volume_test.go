package volume

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

func TestListFlags(t *testing.T) {
	op := &listOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"-p", "2", "-l", "10", "-f", "json"}))

	assert.Equal(t, 2, op.list.Page)
	assert.Equal(t, 10, op.list.Limit)
	assert.Equal(t, list.FormatJSON, op.list.Format)
}

func TestListDefaults(t *testing.T) {
	op := &listOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(nil))

	assert.Equal(t, 1, op.list.Page)
	assert.Equal(t, 0, op.list.Limit)
	assert.Equal(t, list.FormatPretty, op.list.Format)
	assert.NoError(t, op.list.Validate())
}

func TestListRejectsAnUnknownFormat(t *testing.T) {
	op := &listOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"-f", "yaml"}))

	assert.Error(t, op.list.Validate())
}

func TestToListedVolume(t *testing.T) {
	row := toListedVolume(api.Volume{VolumeID: "vol-1", Name: "data"})

	assert.Equal(t, listedVolume{VolumeID: "vol-1", Name: "data"}, row)
}

func TestCommandsAreRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Command().Commands() {
		names[c.Name()] = true
	}

	for _, name := range []string{"create", "get", "list", "delete"} {
		assert.True(t, names[name], "volume %s is missing", name)
	}
}

func TestDeleteNeedsAtLeastOneVolume(t *testing.T) {
	op := &deleteOperation{}
	c := op.Command()

	assert.Error(t, c.Args(c, nil))
	assert.NoError(t, c.Args(c, []string{"vol-1"}))
}
