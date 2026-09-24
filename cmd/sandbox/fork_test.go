package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseForkFlags registers the fork flags the way the command does and returns
// the operation they filled in.
func parseForkFlags(t *testing.T, args ...string) *forkOperation {
	t.Helper()

	op := &forkOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))

	return op
}

func TestForkFlagsStayUnsetByDefault(t *testing.T) {
	op := parseForkFlags(t, "sbx-1")

	assert.Nil(t, op.req.Count, "an absent --count leaves the platform's default")
	assert.Nil(t, op.req.Timeout, "an absent --timeout leaves the platform's default")
	assert.False(t, op.dryRun)
}

func TestForkFlags(t *testing.T) {
	op := parseForkFlags(t, "--count", "3", "--timeout", "60", "--dry-run", "sbx-1")

	require.NotNil(t, op.req.Count)
	assert.Equal(t, int32(3), *op.req.Count)

	require.NotNil(t, op.req.Timeout)
	assert.Equal(t, int32(60), *op.req.Timeout)

	assert.True(t, op.dryRun)
}

func TestForkRejectsNonNumericCount(t *testing.T) {
	op := &forkOperation{}
	c := op.Command()
	c.SetOutput(nil)

	assert.Error(t, c.ParseFlags([]string{"--count", "many"}))
}
