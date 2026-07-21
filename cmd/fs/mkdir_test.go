package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMkdirCommand(t *testing.T) {
	cmd := newMkdirCmd()

	assert.Equal(t, "mkdir <sandbox-id> <dir>", cmd.Use)
	require.NoError(t, cmd.Args(cmd, []string{"sandbox-id", "/home/user/site"}))
	require.Error(t, cmd.Args(cmd, []string{"sandbox-id"}))
	require.Error(t, cmd.Args(cmd, []string{"sandbox-id", "/home/user/site", "extra"}))
}

func TestFsCommandIncludesMkdir(t *testing.T) {
	cmd, _, err := NewFsCmd().Find([]string{"mkdir"})

	require.NoError(t, err)
	assert.Equal(t, "mkdir", cmd.Name())
}
