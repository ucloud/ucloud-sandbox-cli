package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

func TestHostFlags(t *testing.T) {
	op := &hostOperation{}
	c := op.Command()

	require.NoError(t, c.ParseFlags(nil))
	assert.False(t, op.url)

	require.NoError(t, c.ParseFlags([]string{"--url"}))
	assert.True(t, op.url)
}

func TestHostArgCount(t *testing.T) {
	c := (&hostOperation{}).Command()

	assert.Error(t, c.Args(c, []string{"sbx-1"}), "the port is required")
	assert.NoError(t, c.Args(c, []string{"sbx-1", "3000"}))
	assert.Error(t, c.Args(c, []string{"sbx-1", "3000", "extra"}))
}

func TestHostRejectsAnInvalidPort(t *testing.T) {
	op := &hostOperation{}

	// The port is checked before anything is connected to, so Run fails
	// without reaching the client.
	for _, port := range []string{"http", "0", "65536", "-1"} {
		err := op.Run(cmd.OperationContext{Args: []string{"sbx-1", port}})
		assert.ErrorContains(t, err, "port", "port %q", port)
	}
}
