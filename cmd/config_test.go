package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UCLOUD_SANDBOX_API_KEY", "abcd12345678wxyz")
	t.Setenv("UCLOUD_SANDBOX_REGION", "cn-sh")
	t.Setenv("UCLOUD_SANDBOX_DOMAIN", "")
	t.Setenv("UCLOUD_SANDBOX_INSECURE_HTTP", "true")

	var output bytes.Buffer
	cmd := NewConfigCmd()
	cmd.SetOut(&output)
	require.NoError(t, cmd.Execute())

	assert.JSONEq(t, `{
		"api_key": "****",
		"region": "cn-sh",
		"insecure_http": true
	}`, output.String())
	assert.NotContains(t, output.String(), "abcd12345678wxyz")
}

func TestConfigCmd_UsesEffectiveConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UCLOUD_SANDBOX_API_KEY", "environment-key")
	t.Setenv("UCLOUD_SANDBOX_REGION", "")
	t.Setenv("UCLOUD_SANDBOX_DOMAIN", "")
	t.Setenv("UCLOUD_SANDBOX_INSECURE_HTTP", "")

	configDir := filepath.Join(home, ".ucloud-sandbox-cli")
	require.NoError(t, os.MkdirAll(configDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
		"api_key": "file-api-key",
		"region": "cn-bj"
	}`), 0600))

	var output bytes.Buffer
	cmd := NewConfigCmd()
	cmd.SetOut(&output)
	require.NoError(t, cmd.Execute())

	assert.Contains(t, output.String(), `"api_key": "****"`)
	assert.Contains(t, output.String(), `"region": "cn-bj"`)
	assert.NotContains(t, output.String(), "environment-key")
	assert.NotContains(t, output.String(), "file-api-key")
}

func TestMaskAPIKey(t *testing.T) {
	assert.Empty(t, maskAPIKey(""))
	assert.Equal(t, "****", maskAPIKey("short"))
	assert.Equal(t, "****", maskAPIKey("abcd1234wxyz"))
}
