package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPauseFlags(t *testing.T) {
	op := &pauseOperation{}
	c := op.Command()

	require.NoError(t, c.ParseFlags(nil))
	assert.Nil(t, op.req.Memory, "an absent --memory leaves the platform's default")

	require.NoError(t, c.ParseFlags([]string{"--memory=false"}))
	require.NotNil(t, op.req.Memory)
	assert.False(t, *op.req.Memory)
}

func TestPauseArgCount(t *testing.T) {
	c := (&pauseOperation{}).Command()

	assert.Error(t, c.Args(c, nil))
	assert.NoError(t, c.Args(c, []string{"sbx-1"}))
	assert.Error(t, c.Args(c, []string{"sbx-1", "extra"}))
}
