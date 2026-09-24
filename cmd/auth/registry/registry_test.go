package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	internalregistry "github.com/ucloud/ucloud-sandbox-cli/internal/registry"
)

func TestCommandsAreRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Command().Commands() {
		names[c.Name()] = true
	}

	for _, name := range []string{"login", "logout"} {
		assert.True(t, names[name], "auth registry %s is missing", name)
	}
}

func TestDomainOf(t *testing.T) {
	domain, err := domainOf(nil)
	require.NoError(t, err)
	assert.Equal(t, internalregistry.DefaultDomain, domain, "no argument means the default registry")

	domain, err = domainOf([]string{"https://GHCR.io/"})
	require.NoError(t, err)
	assert.Equal(t, "ghcr.io", domain)

	_, err = domainOf([]string{"ghcr.io/owner/app"})
	assert.Error(t, err, "an image reference is not a registry")
}

func TestLogoutRemovesOneRegistry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	require.NoError(t, config.Save(&config.Config{
		APIKey: "keep-me",
		Registries: map[string]config.RegistryAuth{
			"ghcr.io":   {Username: "a", Password: "a-pass"},
			"docker.io": {Username: "b", Password: "b-pass"},
		},
	}))

	op := &logoutOperation{}
	require.NoError(t, op.Run(cmd.OperationContext{Args: []string{"ghcr.io"}}))

	cfg, err := config.LoadFile()
	require.NoError(t, err)

	assert.NotContains(t, cfg.Registries, "ghcr.io")
	assert.Contains(t, cfg.Registries, "docker.io", "the other registry is left alone")
	assert.Equal(t, "keep-me", cfg.APIKey, "the rest of the config survives")
}

func TestLogoutOfAnUnknownRegistryIsNotAFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	op := &logoutOperation{}

	// Nothing has been saved at all, so there is not even a config file.
	assert.NoError(t, op.Run(cmd.OperationContext{Args: []string{"ghcr.io"}}))
}

func TestLogoutNormalizesTheDomain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	require.NoError(t, config.Save(&config.Config{
		Registries: map[string]config.RegistryAuth{"ghcr.io": {Username: "a"}},
	}))

	op := &logoutOperation{}
	require.NoError(t, op.Run(cmd.OperationContext{Args: []string{"https://GHCR.io"}}))

	cfg, err := config.LoadFile()
	require.NoError(t, err)
	assert.Empty(t, cfg.Registries)
}

func TestLogoutRejectsAnImageReference(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	op := &logoutOperation{}
	assert.Error(t, op.Run(cmd.OperationContext{Args: []string{"ghcr.io/owner/app"}}))
}

func TestLoginArgCount(t *testing.T) {
	c := (&loginOperation{}).Command()

	assert.NoError(t, c.Args(c, nil), "the domain defaults to the public registry")
	assert.NoError(t, c.Args(c, []string{"ghcr.io"}))
	assert.Error(t, c.Args(c, []string{"ghcr.io", "extra"}))
}
