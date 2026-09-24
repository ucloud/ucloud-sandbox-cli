package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		wantID    string
		wantPath  string
		wantError bool
	}{
		{name: "local relative", arg: "file.txt", wantPath: "file.txt"},
		{name: "local absolute", arg: "/tmp/file.txt", wantPath: "/tmp/file.txt"},
		{name: "local windows drive", arg: `C:\data\file.txt`, wantPath: `C:\data\file.txt`},
		{name: "remote", arg: "sbx-123:/home/user/file.txt", wantID: "sbx-123", wantPath: "/home/user/file.txt"},
		{name: "remote relative path", arg: "sbx-123:file.txt", wantID: "sbx-123", wantPath: "file.txt"},
		{name: "remote directory destination", arg: "sbx-123:/home/user/", wantID: "sbx-123", wantPath: "/home/user/"},
		{name: "remote missing path", arg: "sbx-123:", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePath(tt.arg)
			if tt.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantID, got.sandboxID)
			assert.Equal(t, tt.wantPath, got.path)
			assert.Equal(t, tt.wantID != "", got.isRemote())
		})
	}
}

func TestCpRejectsPathPairsWithoutExactlyOneSandbox(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "both remote",
			args:    []string{"sbx-1:/a", "sbx-2:/b"},
			wantErr: "at most one path may include a sandbox id",
		},
		{
			name:    "neither remote",
			args:    []string{"/a", "/b"},
			wantErr: "exactly one path must include a sandbox id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &cpOperation{}
			c := op.Command()

			// The pairing is checked before anything is connected to, so Run
			// fails without reaching the client.
			err := op.Run(cmd.OperationContext{Cmd: c, Args: tt.args})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
