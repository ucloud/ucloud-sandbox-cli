package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseConnectFlags registers the connect flags the way the command does and
// returns the operation they filled in.
func parseConnectFlags(t *testing.T, args ...string) *connectOperation {
	t.Helper()

	op := &connectOperation{}
	require.NoError(t, op.Command().ParseFlags(args))

	return op
}

func TestConnectFlagDefaults(t *testing.T) {
	op := parseConnectFlags(t)

	assert.Zero(t, op.req.Timeout)
	assert.Nil(t, op.req.Memory)
	assert.False(t, op.detached)
	assert.Empty(t, op.user)
	assert.Empty(t, op.commandsOptions.Cwd)
	assert.Nil(t, op.commandsOptions.EnvVars)
}

func TestConnectFlags(t *testing.T) {
	op := parseConnectFlags(t,
		"--timeout", "60",
		"--memory=false",
		"--detached",
		"-u", "root",
		"-c", "/home/user",
		"-e", "FOO: bar",
		"--env", "BAZ:qux",
	)

	assert.Equal(t, int32(60), op.req.Timeout)
	require.NotNil(t, op.req.Memory)
	assert.False(t, *op.req.Memory)
	assert.True(t, op.detached)
	assert.Equal(t, "root", op.user)
	assert.Equal(t, "/home/user", op.commandsOptions.Cwd)
	assert.Equal(t, map[string]string{"FOO": "bar", "BAZ": "qux"}, op.commandsOptions.EnvVars)
}
