package tag

import (
	"testing"
	"time"

	"github.com/google/uuid"
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

	for _, name := range []string{"assign", "list", "remove"} {
		assert.True(t, names[name], "template tag %s is missing", name)
	}
}

func TestAssignArgCount(t *testing.T) {
	c := (&assignOperation{}).Command()

	assert.Error(t, c.Args(c, nil), "the target build is required")
	assert.Error(t, c.Args(c, []string{"demo"}), "at least one tag is required")
	assert.NoError(t, c.Args(c, []string{"demo:latest", "v1"}))
	assert.NoError(t, c.Args(c, []string{"demo", "v1", "v2"}))
}

func TestRemoveArgCount(t *testing.T) {
	c := (&removeOperation{}).Command()

	assert.Error(t, c.Args(c, nil))
	assert.Error(t, c.Args(c, []string{"demo"}))
	assert.NoError(t, c.Args(c, []string{"demo", "v1"}))
}

func TestRemoveFlags(t *testing.T) {
	op := &removeOperation{}
	c := op.Command()

	require.NoError(t, c.ParseFlags(nil))
	assert.False(t, op.yes)

	require.NoError(t, c.ParseFlags([]string{"-y"}))
	assert.True(t, op.yes)
}

func TestListArgCount(t *testing.T) {
	c := (&listOperation{}).Command()

	assert.Error(t, c.Args(c, nil))
	assert.NoError(t, c.Args(c, []string{"demo"}))
	assert.Error(t, c.Args(c, []string{"demo", "extra"}))
}

func TestListFlags(t *testing.T) {
	op := &listOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"-f", "json"}))

	assert.Equal(t, list.FormatJSON, op.list.Format)
}

func TestToListedTag(t *testing.T) {
	created := time.Date(2026, time.July, 17, 14, 30, 0, 0, time.Local)
	buildID := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	row := toListedTag(api.TemplateTag{Tag: "v1", BuildID: buildID, CreatedAt: created})

	assert.Equal(t, listedTag{
		Tag:       "v1",
		BuildID:   buildID.String(),
		CreatedAt: created,
	}, row)
}
