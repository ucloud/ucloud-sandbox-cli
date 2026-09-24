package secret

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

func TestCreateFlags(t *testing.T) {
	op := &createOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{
		"--value", "s3cr3t",
		"-m", "owner: platform",
		"-m", "env: prod",
	}))

	require.NotNil(t, op.req.Metadata)
	assert.Equal(t, api.SecretMetadata{"owner": "platform", "env": "prod"}, *op.req.Metadata)

	value, err := op.value.Read("Value")
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", value)
}

func TestCreateMetadataStaysUnsetByDefault(t *testing.T) {
	op := &createOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(nil))

	assert.Nil(t, op.req.Metadata)
}

func TestUpdateFlags(t *testing.T) {
	op := &updateOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"--value", "next", "-m", "rotated: 2026-09-24"}))

	require.NotNil(t, op.req.Metadata)
	assert.Equal(t, api.SecretMetadata{"rotated": "2026-09-24"}, *op.req.Metadata)

	value, err := op.value.Read("Value")
	require.NoError(t, err)
	assert.Equal(t, "next", value)
}

func TestListFlags(t *testing.T) {
	op := &listOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"-p", "2", "-l", "10", "-f", "json"}))

	assert.Equal(t, 2, op.list.Page)
	assert.Equal(t, 10, op.list.Limit)
	assert.Equal(t, list.FormatJSON, op.list.Format)

	// The endpoint's own cursor and page size are not flags.
	assert.Nil(t, c.Flags().Lookup("next-token"))
	assert.Nil(t, op.params.NextToken)
	assert.Nil(t, op.params.Limit)
}

func TestToListedSecret(t *testing.T) {
	created := time.Date(2026, time.July, 17, 14, 30, 0, 0, time.Local)

	row := toListedSecret(api.Secret{
		SecretID:       "sec_1",
		Name:           "openai-key",
		CurrentVersion: 3,
		CreatedAt:      created,
		UpdatedAt:      created.Add(time.Hour),
		Metadata:       api.SecretMetadata{"owner": "platform"},
	})

	assert.Equal(t, listedSecret{
		SecretID:       "sec_1",
		Name:           "openai-key",
		CurrentVersion: 3,
		CreatedAt:      created,
		UpdatedAt:      created.Add(time.Hour),
	}, row)
}

func TestCommandsAreRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Command().Commands() {
		names[c.Name()] = true
	}

	for _, name := range []string{"create", "get", "list", "update", "delete"} {
		assert.True(t, names[name], "secret %s is missing", name)
	}
}
