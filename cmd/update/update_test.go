package update

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/internal/selfupdate"
)

func TestDownloadProgress(t *testing.T) {
	payload := bytes.Repeat([]byte("archive"), 32*1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)
	client := &selfupdate.Client{HTTPClient: server.Client()}
	asset := selfupdate.Asset{
		Name:        "ucloud-sandbox-cli_linux_amd64.tar.gz",
		DownloadURL: server.URL,
		Size:        int64(len(payload)),
	}

	for _, interactive := range []bool{false, true} {
		t.Run(fmt.Sprintf("interactive=%t", interactive), func(t *testing.T) {
			var out bytes.Buffer
			got, err := downloadWithProgress(context.Background(), client, asset, &out, interactive)
			require.NoError(t, err)
			assert.Equal(t, payload, got)

			header := "Downloading " + asset.Name + "...\n"
			require.True(t, strings.HasPrefix(out.String(), header))
			progress := strings.TrimPrefix(out.String(), header)
			if !interactive {
				assert.Empty(t, progress, "redirected output should contain no redraws or control codes")
				return
			}

			assert.Equal(t, 1, strings.Count(progress, "\n"), "only completion should add a newline")
			assert.Equal(t, 1, strings.Count(progress, "100%"))
			assert.True(t, strings.HasSuffix(progress, "\n"))
			frames := strings.Split(strings.TrimSuffix(progress, "\n"), "\r")
			require.GreaterOrEqual(t, len(frames), 3, "exercise both an intermediate and a completed frame")
			for _, frame := range frames {
				visible := strings.ReplaceAll(frame, "\x1b[0K", "")
				assert.Less(t, utf8.RuneCountInString(visible), 80, "progress must fit an 80-column terminal: %q", visible)
			}
		})
	}
}

func TestDownloadProgressError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100000")
		_, _ = w.Write(bytes.Repeat([]byte("x"), 1024))
	}))
	t.Cleanup(server.Close)
	client := &selfupdate.Client{HTTPClient: server.Client()}
	asset := selfupdate.Asset{Name: "archive.tar.gz", DownloadURL: server.URL, Size: 100000}

	var out bytes.Buffer
	got, err := downloadWithProgress(context.Background(), client, asset, &out, true)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.NotContains(t, out.String(), "100%")
	assert.True(t, strings.HasSuffix(out.String(), "\n"), "errors should start below the progress bar")
}
