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
	for _, k := range []string{envAPIKey, envRegion, envDomain, envInsecureHTTP, envRegistries} {
		t.Setenv(k, "")
	}
}

func TestLoad_FileOnly(t *testing.T) {
	home := setupHome(t)
	clearEnv(t)
	writeConfig(t, home, &Config{APIKey: "key1", Region: "cn-sh", InsecureHTTP: true,
		Registries: map[string]RegistryAuth{"docker.io": {Username: "file-user", Password: "file-pass"}}})

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "key1", cfg.APIKey)
	assert.Equal(t, "cn-sh", cfg.Region)
	assert.True(t, cfg.InsecureHTTP)
	assert.Equal(t, map[string]RegistryAuth{"docker.io": {Username: "file-user", Password: "file-pass"}}, cfg.Registries)
}

func TestRegistryAuth(t *testing.T) {
	cfg := &Config{Registries: map[string]RegistryAuth{
		"uhub.service.ucloud.cn": {Username: "builder", Password: "s3cr3t"},
	}}

	auth, ok := cfg.RegistryAuth("uhub.service.ucloud.cn")
	require.True(t, ok)
	assert.Equal(t, "builder", auth.Username)

	// Domains are stored lower-cased, so a lookup is too.
	auth, ok = cfg.RegistryAuth("UHub.Service.UCloud.CN")
	require.True(t, ok)
	assert.Equal(t, "builder", auth.Username)

	_, ok = cfg.RegistryAuth("docker.io")
	assert.False(t, ok)
}

func TestRegistryAuthOnAnEmptyConfig(t *testing.T) {
	_, ok := (&Config{}).RegistryAuth("docker.io")
	assert.False(t, ok)
}

func TestLoad_EnvRegistries(t *testing.T) {
	home := setupHome(t)
	clearEnv(t)
	writeConfig(t, home, &Config{
		Registries: map[string]RegistryAuth{"docker.io": {Username: "file-user"}}})
	t.Setenv(envRegistries, `{"ghcr.io": {"username": "env-user", "password": "env-pass"}}`)

	cfg, err := Load()
	require.NoError(t, err)

	// The environment replaces the file's map rather than merging into it.
	assert.Equal(t, map[string]RegistryAuth{
		"ghcr.io": {Username: "env-user", Password: "env-pass"},
	}, cfg.Registries)
}

func TestLoad_InvalidEnvRegistries(t *testing.T) {
	setupHome(t)
	clearEnv(t)
	t.Setenv(envRegistries, "not json")

	_, err := Load()
	assert.ErrorContains(t, err, envRegistries)
}

func TestLoad_EnvOverride(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, &Config{APIKey: "file-key", Region: "file-region", InsecureHTTP: true})
	t.Setenv(envAPIKey, "env-key")
	t.Setenv(envRegion, "env-region")
	t.Setenv(envDomain, "env.example.com")
	t.Setenv(envInsecureHTTP, "false")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "env-key", cfg.APIKey)
	assert.Equal(t, "env-region", cfg.Region)
	assert.Equal(t, "env.example.com", cfg.Domain)
	assert.False(t, cfg.InsecureHTTP)
}

func TestLoad_EnvInsecureHTTPTrue(t *testing.T) {
	setupHome(t)
	clearEnv(t)
	t.Setenv(envInsecureHTTP, "true")

	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.InsecureHTTP)
}

func TestLoad_InvalidEnvInsecureHTTP(t *testing.T) {
	setupHome(t)
	clearEnv(t)
	t.Setenv(envInsecureHTTP, "not-a-bool")

	_, err := Load()
	assert.ErrorContains(t, err, envInsecureHTTP)
}

func TestLoad_NoFile(t *testing.T) {
	setupHome(t)
	clearEnv(t)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.APIKey)
	assert.Empty(t, cfg.Region)
	assert.Empty(t, cfg.Domain)
	assert.False(t, cfg.InsecureHTTP)
}

func TestLoadFile_IgnoresTheEnvironment(t *testing.T) {
	home := setupHome(t)
	writeConfig(t, home, &Config{APIKey: "file-key", Region: "file-region"})
	t.Setenv(envAPIKey, "env-key")
	t.Setenv(envRegion, "env-region")

	// A read-modify-write must not persist what only the environment said.
	cfg, err := LoadFile()
	require.NoError(t, err)
	assert.Equal(t, "file-key", cfg.APIKey)
	assert.Equal(t, "file-region", cfg.Region)
}

func TestLoadFile_NoFile(t *testing.T) {
	setupHome(t)

	cfg, err := LoadFile()
	require.NoError(t, err)
	assert.Equal(t, &Config{}, cfg)
}

func TestLoadFile_Invalid(t *testing.T) {
	home := setupHome(t)
	dir := filepath.Join(home, configDir)
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, configFile), []byte("not json"), 0600))

	_, err := LoadFile()
	assert.ErrorContains(t, err, "parse config")
}

func TestPath(t *testing.T) {
	home := setupHome(t)

	path, err := Path()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, configDir, configFile), path)
}

func TestSave(t *testing.T) {
	home := setupHome(t)

	in := &Config{APIKey: "save-key", Region: "cn-bj", InsecureHTTP: true,
		Registries: map[string]RegistryAuth{"ghcr.io": {Username: "save-user", Password: "save-pass"}}}
	require.NoError(t, Save(in))

	data, err := os.ReadFile(filepath.Join(home, configDir, configFile))
	require.NoError(t, err)
	var out Config
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, in.APIKey, out.APIKey)
	assert.Equal(t, in.Region, out.Region)
	assert.Equal(t, in.InsecureHTTP, out.InsecureHTTP)
	assert.Equal(t, in.Registries, out.Registries)
	assert.Contains(t, string(data), `"insecure_http": true`)
	assert.NotContains(t, string(data), `"insecure":`)
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
