package volume

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/api"
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
	row := toListedVolume(api.Volume{VolumeID: uuid.MustParse(testID), Name: "data"})

	assert.Equal(t, listedVolume{VolumeID: testID, Name: "data", TotalSizeGiB: "-", UsedSizeGiB: "-", AvailableSizeGiB: "-"}, row)
}

func TestCommandsAreRegistered(t *testing.T) {
	root := Command()
	for _, name := range []string{"create", "get", "list", "delete", "resize", "content"} {
		c, remaining, err := root.Find([]string{name})
		require.NoError(t, err)
		assert.Empty(t, remaining)
		assert.Equal(t, name, c.Name())
	}
	for _, group := range []string{"content", "ct"} {
		for _, name := range []string{"upload", "download", "list", "ls", "mkdir", "remove", "rm", "stat", "rename", "mv", "connect"} {
			c, remaining, err := root.Find([]string{group, name})
			require.NoError(t, err)
			assert.Empty(t, remaining)
			assert.Equal(t, "volume content", c.Parent().CommandPath())
		}
	}
	for _, name := range []string{"v2", "fs"} {
		_, _, err := root.Find([]string{name})
		assert.Error(t, err)
	}
}

func TestDeleteNeedsAtLeastOneVolume(t *testing.T) {
	op := &deleteOperation{}
	c := op.Command()

	assert.Error(t, c.Args(c, nil))
	assert.NoError(t, c.Args(c, []string{"vol-1"}))
}

func TestCapacityZeroIsNotMissing(t *testing.T) {
	row := toListedVolume(api.Volume{VolumeID: uuid.MustParse(testID), Name: "data",
		TotalSizeGiB: new(int64(1)), UsedSizeGiB: new(float64(0)), AvailableSizeGiB: new(float64(1)),
	})
	assert.Equal(t, "1", row.TotalSizeGiB)
	assert.Equal(t, "0", row.UsedSizeGiB)
	assert.Equal(t, "1", row.AvailableSizeGiB)
}
