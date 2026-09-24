package secret

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseValueFlags registers the value flags and returns the source they filled
// in.
func parseValueFlags(t *testing.T, args ...string) *valueSource {
	t.Helper()

	source := &valueSource{}

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.SetOutput(nil)
	source.AddFlags(fs)

	require.NoError(t, fs.Parse(args))

	return source
}

// withStdin points os.Stdin at a file holding content for the duration of the
// test.
func withStdin(t *testing.T, content string) {
	t.Helper()

	read, write, err := os.Pipe()
	require.NoError(t, err)

	_, err = write.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, write.Close())

	original := os.Stdin
	os.Stdin = read
	t.Cleanup(func() {
		os.Stdin = original
		read.Close()
	})
}

func TestValueSourceFromFlag(t *testing.T) {
	source := parseValueFlags(t, "--value", "s3cr3t")

	value, err := source.Read("Value")
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", value)
}

func TestValueSourceFromFlagKeepsAnExplicitlyEmptyValue(t *testing.T) {
	source := parseValueFlags(t, "--value", "")

	// An explicitly empty --value is a value, not a missing one, so nothing is
	// asked for.
	value, err := source.Read("Value")
	require.NoError(t, err)
	assert.Empty(t, value)
}

func TestValueSourceFromStdin(t *testing.T) {
	withStdin(t, "s3cr3t\n")

	source := parseValueFlags(t, "--value-stdin")

	value, err := source.Read("Value")
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", value, "the newline that ended the input is not part of the secret")
}

func TestValueSourceFromStdinKeepsInnerNewlines(t *testing.T) {
	withStdin(t, "-----BEGIN KEY-----\nabc\n-----END KEY-----\n")

	source := parseValueFlags(t, "--value-stdin")

	value, err := source.Read("Value")
	require.NoError(t, err)
	assert.Equal(t, "-----BEGIN KEY-----\nabc\n-----END KEY-----", value)
}

func TestValueSourceRejectsBothFlags(t *testing.T) {
	source := parseValueFlags(t, "--value", "s3cr3t", "--value-stdin")

	_, err := source.Read("Value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestValueSourceWithoutATerminalNeedsAFlag(t *testing.T) {
	// A pipe is not a terminal, so there is nobody to ask.
	withStdin(t, "")

	source := parseValueFlags(t)

	_, err := source.Read("Value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--value-stdin")
}
