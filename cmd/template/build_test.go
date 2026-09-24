package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// templateDir writes a template directory holding a config and a Dockerfile.
func templateDir(t *testing.T, root, name string, cfg *LocalConfig, dockerfiles ...string) string {
	t.Helper()

	dir := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(dir, 0755))

	if cfg != nil {
		require.NoError(t, saveConfig(dir, cfg))
	}

	for _, filename := range dockerfiles {
		require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte("FROM alpine\n"), 0644))
	}

	return dir
}

func parseBuildFlags(t *testing.T, args ...string) *buildOperation {
	t.Helper()

	op := &buildOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))

	return op
}

func TestBuildFlags(t *testing.T) {
	op := parseBuildFlags(t,
		"-p", "/tmp/projects",
		"-d", "custom.dockerfile",
		"--cmd", "/start.sh",
		"--ready-cmd", "curl -f localhost:8080",
		"--cpu-count", "4",
		"--memory-mb", "2048",
		"--min-free-disk-mb", "512",
		"-t", "v1", "-t", "latest",
		"--no-cache", "--publish",
		"--level", "debug",
	)

	assert.Equal(t, "/tmp/projects", op.path)
	assert.Equal(t, "custom.dockerfile", op.dockerfile)
	assert.Equal(t, "/start.sh", op.startCmd)
	assert.Equal(t, "curl -f localhost:8080", op.readyCmd)

	require.NotNil(t, op.req.CpuCount)
	assert.Equal(t, int32(4), *op.req.CpuCount)
	require.NotNil(t, op.req.MemoryMB)
	assert.Equal(t, int32(2048), *op.req.MemoryMB)
	require.NotNil(t, op.req.MinFreeDiskMb)
	assert.Equal(t, int32(512), *op.req.MinFreeDiskMb)
	require.NotNil(t, op.req.Tags)
	assert.Equal(t, []string{"v1", "latest"}, *op.req.Tags)

	assert.True(t, op.noCache)
	assert.True(t, op.publish)
	assert.Equal(t, "debug", op.logLevel)
}

func TestBuildResourceFlagsStayUnsetByDefault(t *testing.T) {
	op := parseBuildFlags(t)

	// nil means "not given", which lets the local config and then the platform
	// decide, instead of the CLI inventing a default.
	assert.Nil(t, op.req.CpuCount)
	assert.Nil(t, op.req.MemoryMB)
	assert.Nil(t, op.req.MinFreeDiskMb)
	assert.Nil(t, op.req.Tags)

	assert.Equal(t, ".", op.path)
	assert.Equal(t, string(api.LogLevelInfo), op.logLevel)
}

func TestValidateResources(t *testing.T) {
	cpu := func(n int32) *int32 { return &n }

	tests := []struct {
		name    string
		req     api.TemplateBuildRequestV3
		wantErr string
	}{
		{name: "everything unset"},
		{name: "valid", req: api.TemplateBuildRequestV3{CpuCount: cpu(2), MemoryMB: cpu(1024)}},
		{name: "zero cpu", req: api.TemplateBuildRequestV3{CpuCount: cpu(0)}, wantErr: "CPU count"},
		{name: "negative cpu", req: api.TemplateBuildRequestV3{CpuCount: cpu(-1)}, wantErr: "CPU count"},
		{name: "zero memory", req: api.TemplateBuildRequestV3{MemoryMB: cpu(0)}, wantErr: "greater than zero"},
		{name: "odd memory", req: api.TemplateBuildRequestV3{MemoryMB: cpu(1025)}, wantErr: "even number"},
		{name: "zero free disk is allowed", req: api.TemplateBuildRequestV3{MinFreeDiskMb: cpu(0)}},
		{name: "negative free disk", req: api.TemplateBuildRequestV3{MinFreeDiskMb: cpu(-1)}, wantErr: "zero or greater"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResources(tt.req)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestResolveBuildContextPrefersTheNamedDirectory(t *testing.T) {
	root := t.TempDir()

	// Both the root and a directory named after the template hold a config.
	require.NoError(t, saveConfig(root, &LocalConfig{TemplateName: "root-one"}))
	templateDir(t, root, "demo", &LocalConfig{TemplateName: "demo"})

	path, cfg := resolveBuildContext("demo", root)
	assert.Equal(t, filepath.Join(root, "demo"), path)
	require.NotNil(t, cfg)
	assert.Equal(t, "demo", cfg.TemplateName)
}

func TestResolveBuildContextFallsBackToTheRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, saveConfig(root, &LocalConfig{TemplateName: "root-one"}))

	path, cfg := resolveBuildContext("demo", root)
	assert.Equal(t, root, path)
	require.NotNil(t, cfg)
	assert.Equal(t, "root-one", cfg.TemplateName)
}

func TestResolveBuildContextWithoutAnyConfig(t *testing.T) {
	root := t.TempDir()

	path, cfg := resolveBuildContext("demo", root)
	assert.Equal(t, root, path)
	assert.Nil(t, cfg)
}

func TestResolveDockerfilePath(t *testing.T) {
	t.Run("the conventional name", func(t *testing.T) {
		dir := templateDir(t, t.TempDir(), "demo", nil, defaultDockerfileName)

		path, err := resolveDockerfilePath(dir, "")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, defaultDockerfileName), path)
	})

	t.Run("the fallback name", func(t *testing.T) {
		dir := templateDir(t, t.TempDir(), "demo", nil, fallbackDockerfileName)

		path, err := resolveDockerfilePath(dir, "")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, fallbackDockerfileName), path)
	})

	t.Run("the conventional name wins over the fallback", func(t *testing.T) {
		dir := templateDir(t, t.TempDir(), "demo", nil, defaultDockerfileName, fallbackDockerfileName)

		path, err := resolveDockerfilePath(dir, "")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, defaultDockerfileName), path)
	})

	t.Run("the local config names one", func(t *testing.T) {
		dir := templateDir(t, t.TempDir(), "demo",
			&LocalConfig{Dockerfile: "custom.dockerfile"}, "custom.dockerfile", defaultDockerfileName)

		path, err := resolveDockerfilePath(dir, "")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "custom.dockerfile"), path)
	})

	t.Run("an explicit name wins over the config", func(t *testing.T) {
		dir := templateDir(t, t.TempDir(), "demo",
			&LocalConfig{Dockerfile: "custom.dockerfile"}, "custom.dockerfile", "other.dockerfile")

		path, err := resolveDockerfilePath(dir, "other.dockerfile")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "other.dockerfile"), path)
	})

	t.Run("nothing to build", func(t *testing.T) {
		dir := templateDir(t, t.TempDir(), "demo", nil)

		_, err := resolveDockerfilePath(dir, "")
		assert.ErrorContains(t, err, "no Dockerfile found")
	})

	t.Run("a named file that is not there", func(t *testing.T) {
		dir := templateDir(t, t.TempDir(), "demo", nil, defaultDockerfileName)

		_, err := resolveDockerfilePath(dir, "missing.dockerfile")
		assert.Error(t, err)
	})
}

// The SDK fixes the file context to the Dockerfile's directory and hashes COPY
// sources against it, so a Dockerfile elsewhere would ship the wrong files.
func TestResolveDockerfilePathRejectsOneOutsideTheContext(t *testing.T) {
	root := t.TempDir()
	dir := templateDir(t, root, "demo", nil, defaultDockerfileName)
	outside := templateDir(t, root, "elsewhere", nil, "other.dockerfile")

	_, err := resolveDockerfilePath(dir, filepath.Join(outside, "other.dockerfile"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build context")
}

func TestResolveDockerfilePathRejectsOneInASubdirectory(t *testing.T) {
	dir := templateDir(t, t.TempDir(), "demo", nil)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docker"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "docker", "Dockerfile"), []byte("FROM alpine\n"), 0644))

	_, err := resolveDockerfilePath(dir, "docker/Dockerfile")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build context")
}

func TestRememberTemplateID(t *testing.T) {
	dir := templateDir(t, t.TempDir(), "demo", &LocalConfig{TemplateName: "demo"})

	local, err := loadConfig(dir)
	require.NoError(t, err)

	rememberTemplateID(dir, local, "demo", "tpl-1")

	saved, err := loadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "tpl-1", saved.TemplateID)
	assert.Equal(t, "demo", saved.TemplateName)
}

func TestRememberTemplateIDWithoutALocalConfig(t *testing.T) {
	dir := t.TempDir()

	// No config to record it in, so nothing is written rather than a file
	// appearing where the user kept none.
	rememberTemplateID(dir, nil, "demo", "tpl-1")

	_, err := loadConfig(dir)
	assert.Error(t, err)
}
