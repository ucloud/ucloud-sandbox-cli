package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseExecFlags registers the exec flags the way the command does and returns
// the operation they filled in, along with the remaining arguments.
func parseExecFlags(t *testing.T, args ...string) (*execOperation, []string) {
	t.Helper()

	op := &execOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))

	return op, c.Flags().Args()
}

func TestExecFlags(t *testing.T) {
	op, args := parseExecFlags(t,
		"--timeout", "60",
		"--command-timeout", "-1",
		"--stdin",
		"-u", "root",
		"-c", "/home/user",
		"-e", "FOO: bar",
		"sbx-1", "ls", "-la",
	)

	assert.Equal(t, int32(60), op.req.Timeout)
	assert.Equal(t, -1, op.commandsOptions.TimeoutSeconds)
	assert.True(t, op.commandsOptions.Stdin)
	assert.Equal(t, "root", op.user)
	assert.Equal(t, "/home/user", op.commandsOptions.Cwd)
	assert.Equal(t, map[string]string{"FOO": "bar"}, op.commandsOptions.EnvVars)

	// Flags after the sandbox ID belong to the remote command.
	assert.Equal(t, []string{"sbx-1", "ls", "-la"}, args)
}

func TestExecFlagsStopAtSandboxID(t *testing.T) {
	op, args := parseExecFlags(t, "sbx-1", "env", "-u", "root")

	assert.Empty(t, op.user, "-u after the sandbox ID is the command's, not ours")
	assert.Equal(t, []string{"sbx-1", "env", "-u", "root"}, args)
}

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "a single part keeps its shell syntax",
			parts: []string{"ls -la | wc -l"},
			want:  "ls -la | wc -l",
		},
		{
			name:  "plain parts are joined as they are",
			parts: []string{"ls", "-la", "/tmp"},
			want:  "ls -la /tmp",
		},
		{
			name:  "a part with spaces stays one argument",
			parts: []string{"echo", "hello world"},
			want:  `echo 'hello world'`,
		},
		{
			name:  "a quote inside a part is escaped",
			parts: []string{"echo", "it's"},
			want:  `echo 'it'"'"'s'`,
		},
		{
			name:  "an empty part stays an empty argument",
			parts: []string{"echo", ""},
			want:  "echo ''",
		},
		{
			name:  "a leading -- is dropped",
			parts: []string{"--", "ls", "-la"},
			want:  "ls -la",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildCommand(tt.parts)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildCommandRejectsEmptyCommand(t *testing.T) {
	_, err := buildCommand(nil)
	assert.Error(t, err)

	_, err = buildCommand([]string{"--"})
	assert.Error(t, err)
}
