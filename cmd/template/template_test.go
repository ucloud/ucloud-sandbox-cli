package template

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

	for _, name := range []string{"build", "delete", "get", "init", "list", "logs", "publish", "tag"} {
		assert.True(t, names[name], "template %s is missing", name)
	}
}

func TestBuildIsAlsoReachableAsCreate(t *testing.T) {
	c, _, err := Command().Find([]string{"create"})
	require.NoError(t, err)
	assert.Equal(t, "build", c.Name())
}

func TestLocalConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()

	in := &LocalConfig{
		TemplateName: "demo",
		TemplateID:   "tpl-1",
		CPUCount:     2,
		MemoryMB:     1024,
		Dockerfile:   "template.dockerfile",
	}
	require.NoError(t, saveConfig(dir, in))

	out, err := loadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, in, out)

	require.NoError(t, deleteConfig(dir))
	_, err = loadConfig(dir)
	assert.Error(t, err)

	// Deleting a config that is not there is what the caller asked for.
	assert.NoError(t, deleteConfig(dir))
}

func TestLoadConfigRejectsBrokenJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), []byte("not json"), 0644))

	_, err := loadConfig(dir)
	assert.ErrorContains(t, err, "parse config")
}

func TestToListedTemplate(t *testing.T) {
	created := time.Date(2026, time.July, 17, 14, 30, 0, 0, time.Local)

	row := toListedTemplate(api.Template{
		TemplateID:  "tpl-1",
		Names:       []string{"team/demo", "demo"},
		BuildStatus: api.TemplateBuildStatusReady,
		Public:      true,
		CpuCount:    2,
		MemoryMB:    2048,
		CreatedAt:   created,
	})

	assert.Equal(t, listedTemplate{
		TemplateID: "tpl-1",
		Names:      "team/demo, demo",
		Status:     "ready",
		Visibility: "Public",
		CPUCount:   2,
		MemoryMB:   2048,
		CreatedAt:  created,
	}, row)
}

func TestToListedTemplateWithoutABuild(t *testing.T) {
	row := toListedTemplate(api.Template{TemplateID: "tpl-1"})

	assert.Equal(t, "-", row.Status, "a template that never built has no status to show")
	assert.Equal(t, "Private", row.Visibility)
	assert.Empty(t, row.Names)
}

func TestListFlags(t *testing.T) {
	op := &listOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"-p", "2", "-l", "10", "-f", "json"}))

	assert.Equal(t, 2, op.list.Page)
	assert.Equal(t, 10, op.list.Limit)
	assert.Equal(t, list.FormatJSON, op.list.Format)
}

func TestTargetsFlags(t *testing.T) {
	op := &deleteOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"-p", "/tmp/projects", "-s", "-y"}))

	assert.Equal(t, "/tmp/projects", op.path)
	assert.True(t, op.pick)
	assert.True(t, op.yes)
}

func TestPublishFlags(t *testing.T) {
	op := &publishOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"--unpublish", "-y"}))

	assert.True(t, op.unpublish)
	assert.True(t, op.yes)
	assert.Equal(t, ".", op.path)
}

func TestGetFlags(t *testing.T) {
	op := &getOperation{}
	c := op.Command()

	require.NoError(t, c.ParseFlags(nil))
	assert.Nil(t, op.params.Limit)

	require.NoError(t, c.ParseFlags([]string{"-l", "50"}))
	require.NotNil(t, op.params.Limit)
	assert.Equal(t, int32(50), *op.params.Limit)
}

func TestInitFlags(t *testing.T) {
	op := &initOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"-p", "/tmp/projects", "--from", "alpine", "--memory-mb", "2048"}))

	assert.Equal(t, "/tmp/projects", op.path)
	assert.Equal(t, "alpine", op.from)
	require.NotNil(t, op.memoryMB)
	assert.Equal(t, int32(2048), *op.memoryMB)
	assert.Nil(t, op.cpuCount)
}
