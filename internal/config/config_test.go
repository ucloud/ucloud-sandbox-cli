package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func writeConfig(t *testing.T, home string, cfg *Config) {
	t.Helper()
	dir := filepath.Join(home, configDir)
	require.NoError(t, os.MkdirAll(dir, 0700))
	data, _ := json.Marshal(cfg)
	require.NoError(t, os.WriteFile(filepath.Join(dir, configFile), data, 0600))
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{envAPIKey, envRegion, envDomain} {
		t.Setenv(k, "")
	}
}

func TestLoad_FileOnly(t *testing.T) {
	home := setupHome(t)
	clearEnv(t)
	writeConfig(t, home, &Config{APIKey: "key1", Region: "cn-sh"})

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "key1", cfg.APIKey)
	assert.Equal(t, "cn-sh", cfg.Region)
}

func TestLoad_EnvOverride(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, &Config{APIKey: "file-key", Region: "file-region"})
	t.Setenv(envAPIKey, "env-key")
	t.Setenv(envRegion, "env-region")
	t.Setenv(envDomain, "env.example.com")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "env-key", cfg.APIKey)
	assert.Equal(t, "env-region", cfg.Region)
	assert.Equal(t, "env.example.com", cfg.Domain)
}

func TestLoad_NoFile(t *testing.T) {
	setupHome(t)
	clearEnv(t)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.APIKey)
	assert.Empty(t, cfg.Region)
	assert.Empty(t, cfg.Domain)
}

func TestSave(t *testing.T) {
	home := setupHome(t)

	in := &Config{APIKey: "save-key", Region: "cn-bj"}
	require.NoError(t, Save(in))

	data, err := os.ReadFile(filepath.Join(home, configDir, configFile))
	require.NoError(t, err)
	var out Config
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, in.APIKey, out.APIKey)
	assert.Equal(t, in.Region, out.Region)
}

func TestResolveDomain(t *testing.T) {
	cases := []struct {
		name   string
		cfg    Config
		domain string
	}{
		{"explicit domain", Config{Domain: "custom.example.com"}, "custom.example.com"},
		{"region only", Config{Region: "cn-sh"}, "cn-sh.sandbox.ucloudai.com"},
		{"default", Config{}, "cn-wlcb.sandbox.ucloudai.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.domain, resolveDomain(&tc.cfg))
		})
	}
}

func TestNewClient_MissingAPIKey(t *testing.T) {
	_, err := NewClient(&Config{})
	assert.Error(t, err)
}

func TestNewClient_OK(t *testing.T) {
	client, err := NewClient(&Config{APIKey: "test-key"})
	require.NoError(t, err)
	assert.NotNil(t, client)
}
