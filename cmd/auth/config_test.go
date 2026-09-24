package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

func TestResolveEditor(t *testing.T) {
	t.Run("the command line wins", func(t *testing.T) {
		t.Setenv("EDITOR", "emacs")
		assert.Equal(t, "nano", resolveEditor([]string{"nano"}))
	})

	t.Run("then $EDITOR", func(t *testing.T) {
		t.Setenv("EDITOR", "emacs")
		assert.Equal(t, "emacs", resolveEditor(nil))
	})

	t.Run("an empty $EDITOR falls back", func(t *testing.T) {
		t.Setenv("EDITOR", "")
		assert.Equal(t, fallbackEditor, resolveEditor(nil))
	})

	t.Run("a blank $EDITOR falls back", func(t *testing.T) {
		t.Setenv("EDITOR", "   ")
		assert.Equal(t, fallbackEditor, resolveEditor(nil))
	})

	t.Run("a blank argument falls back", func(t *testing.T) {
		t.Setenv("EDITOR", "")
		assert.Equal(t, fallbackEditor, resolveEditor([]string{"  "}))
	})
}

func TestWriteMaskedConfig(t *testing.T) {
	var out strings.Builder

	require.NoError(t, writeMaskedConfig(&out, &config.Config{
		APIKey:       "secret-key",
		Region:       "cn-sh",
		Domain:       "cn-sh.sandbox.ucloudai.com",
		InsecureHTTP: true,
		Registries: map[string]config.RegistryAuth{
			"docker.io": {Username: "builder", Password: "secret-password"},
			"ghcr.io":   {Username: "other", Password: "other-password"},
		},
	}))

	var shown map[string]any
	require.NoError(t, json.Unmarshal([]byte(out.String()), &shown))

	assert.Equal(t, maskedValue, shown["api_key"])

	// Everything that is not a secret is shown as it is.
	assert.Equal(t, "cn-sh", shown["region"])
	assert.Equal(t, "cn-sh.sandbox.ucloudai.com", shown["domain"])
	assert.Equal(t, true, shown["insecure_http"])

	// Every registry's password is masked, not just the first one.
	registries := shown["registries"].(map[string]any)
	require.Len(t, registries, 2)
	for domain, entry := range registries {
		auth := entry.(map[string]any)
		assert.Equal(t, maskedValue, auth["password"], "registry %s", domain)
		assert.NotEqual(t, maskedValue, auth["username"], "registry %s", domain)
	}

	assert.NotContains(t, out.String(), "secret-key")
	assert.NotContains(t, out.String(), "secret-password")
	assert.NotContains(t, out.String(), "other-password")
}

func TestWriteMaskedConfigShowsEveryField(t *testing.T) {
	var out strings.Builder

	require.NoError(t, writeMaskedConfig(&out, &config.Config{}))

	var shown map[string]any
	require.NoError(t, json.Unmarshal([]byte(out.String()), &shown))

	// An unset secret stays unset rather than being masked, so the output says
	// what is missing.
	assert.Equal(t, "", shown["api_key"])
	assert.Nil(t, shown["registries"])

	for _, field := range []string{
		"api_key", "region", "domain", "insecure_http", "registries",
	} {
		assert.Contains(t, shown, field)
	}
}

func TestWriteMaskedConfigLeavesTheCallersConfigAlone(t *testing.T) {
	cfg := &config.Config{
		APIKey:     "secret-key",
		Registries: map[string]config.RegistryAuth{"ghcr.io": {Password: "secret-password"}},
	}

	var out strings.Builder
	require.NoError(t, writeMaskedConfig(&out, cfg))

	assert.Equal(t, "secret-key", cfg.APIKey)

	// The shallow copy shares the map, so masking must not write into it.
	assert.Equal(t, "secret-password", cfg.Registries["ghcr.io"].Password)
}

func TestConfigRejectsAnEditorWithoutEdit(t *testing.T) {
	op := &configOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(nil))

	err := op.Run(cmd.OperationContext{Cmd: c, Args: []string{"nano"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--edit")
}

func TestEditConfigCreatesTheFileFirst(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// A script standing in for the editor: it records the path it was handed.
	marker := filepath.Join(home, "opened")
	editor := filepath.Join(home, "fake-editor")
	require.NoError(t, os.WriteFile(editor,
		[]byte("#!/bin/sh\nprintf '%s' \"$1\" > "+marker+"\n"), 0700))

	require.NoError(t, editConfig(editor))

	path := filepath.Join(home, ".ucloud-sandbox-cli", "config.json")

	opened, err := os.ReadFile(marker)
	require.NoError(t, err)
	assert.Equal(t, path, string(opened), "the editor is handed the config file")

	// The file the editor opened is valid JSON listing every setting.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"api_key"`)
	assert.Contains(t, string(data), `"registries"`)
}

func TestEditConfigKeepsAnExistingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	require.NoError(t, config.Save(&config.Config{APIKey: "keep-me", Region: "cn-sh"}))

	require.NoError(t, editConfig("true")) // a no-op "editor"

	cfg, err := config.LoadFile()
	require.NoError(t, err)
	assert.Equal(t, "keep-me", cfg.APIKey)
	assert.Equal(t, "cn-sh", cfg.Region)
}

func TestEditConfigReportsAnUnparseableResult(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".ucloud-sandbox-cli", "config.json")

	// An "editor" that leaves broken JSON behind.
	editor := filepath.Join(home, "breaking-editor")
	require.NoError(t, os.WriteFile(editor,
		[]byte("#!/bin/sh\nprintf 'not json' > \"$1\"\n"), 0700))

	err := editConfig(editor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
	assert.Contains(t, err.Error(), "no longer parses")
}

func TestEditConfigReportsAMissingEditor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := editConfig("no-such-editor-exists")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no-such-editor-exists")
}
