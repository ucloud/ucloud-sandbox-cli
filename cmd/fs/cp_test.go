package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		wantID    string
		wantPath  string
		wantError bool
	}{
		{name: "local relative", arg: "file.txt", wantID: "", wantPath: "file.txt"},
		{name: "local absolute", arg: "/tmp/file.txt", wantID: "", wantPath: "/tmp/file.txt"},
		{name: "local windows drive", arg: "C:\\data\\file.txt", wantID: "", wantPath: "C:\\data\\file.txt"},
		{name: "remote", arg: "sbx-123:/home/user/file.txt", wantID: "sbx-123", wantPath: "/home/user/file.txt"},
		{name: "remote relative path", arg: "sbx-123:file.txt", wantID: "sbx-123", wantPath: "file.txt"},
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
