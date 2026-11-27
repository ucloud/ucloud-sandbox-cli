package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectSession(t *testing.T) {
	op := &connectOperation{}
	var requests []string
	ctx, out := operationContext(t, op, []string{testID}, func(r *http.Request) (int, string) {
		endpoint := strings.TrimPrefix(r.URL.Path, contentPrefix)
		value := r.URL.Query().Get("path")
		if r.Method == http.MethodPost {
			var body struct{ Path string }
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			value = body.Path
		}
		requests = append(requests, fmt.Sprintf("%s %s %s", r.Method, endpoint, value))
		switch endpoint {
		case "/paths":
			if r.Method == http.MethodDelete {
				return http.StatusNoContent, ""
			}
			assert.Equal(t, "true", r.URL.Query().Get("followSymlinks"))
			return http.StatusOK, strings.Replace(entryJSON, `"type":"file"`, `"type":"directory"`, 1)
		case "/directories":
			if r.Method == http.MethodPost {
				return http.StatusOK, entryJSON
			}
			assert.Equal(t, "1", r.URL.Query().Get("depth"))
			if value != "." {
				return http.StatusOK, "[]"
			}
			return http.StatusOK, `[{"name":"xxx"},{"name":"dir0"}]`
		case "/files":
			return http.StatusOK, "hello world\n"
		default:
			t.Fatalf("unexpected endpoint: %s", endpoint)
			return 0, ""
		}
	})
	errs := new(bytes.Buffer)
	ctx.Cmd.SetErr(errs)
	ctx.Cmd.SetIn(strings.NewReader("ls\ncd dir0\ncat file.0\nmkdir \"new dir\"\nls 'new dir'\nll 'new dir'\nrm new\\ dir\ncd /\ncd ..\nexit\nmkdir ignored\n"))
	require.NoError(t, op.Run(ctx))
	assert.Empty(t, errs.String())
	assert.Equal(t, "/ > dir0  xxx\n/ > /dir0 > hello world\n/dir0 > /dir0 > /dir0 > /dir0 > /dir0 > / > / > ", out.String())
	assert.Equal(t, []string{
		"GET /paths .", "GET /directories .", "GET /paths dir0", "GET /files dir0/file.0",
		"POST /directories dir0/new dir", "GET /directories dir0/new dir", "GET /directories dir0/new dir", "DELETE /paths dir0/new dir",
		"GET /paths .", "GET /paths .",
	}, requests)
}

func TestConnectErrorsKeepCurrentDirectory(t *testing.T) {
	op := &connectOperation{}
	var listedPaths []string
	ctx, out := operationContext(t, op, []string{testID}, func(r *http.Request) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/directories") {
			listedPaths = append(listedPaths, r.URL.Query().Get("path"))
			return http.StatusOK, "[]"
		}
		switch r.URL.Query().Get("path") {
		case ".", "dir":
			return http.StatusOK, strings.Replace(entryJSON, `"type":"file"`, `"type":"directory"`, 1)
		case "dir/file":
			return http.StatusOK, entryJSON
		default:
			return http.StatusNotFound, `{"code":404,"message":"missing"}`
		}
	})
	errs := new(bytes.Buffer)
	ctx.Cmd.SetErr(errs)
	ctx.Cmd.SetIn(strings.NewReader("cd dir\ncd file\ncd missing\ncat missing\nmkdir\nls a b\nunknown\ncat 'unfinished\nls\ncd\nls\nexit\n"))
	require.NoError(t, op.Run(ctx))
	assert.Contains(t, errs.String(), "not a directory")
	assert.Contains(t, errs.String(), "missing")
	assert.Contains(t, errs.String(), "usage: mkdir <path>")
	assert.Contains(t, errs.String(), "usage: ls [path]")
	assert.Contains(t, errs.String(), "supported commands:")
	assert.Contains(t, errs.String(), "unfinished quote")
	assert.Equal(t, []string{"dir", "."}, listedPaths)
	assert.Contains(t, out.String(), "/dir > ")
}

func TestConnectDistinguishesAbsoluteAndRelativePaths(t *testing.T) {
	op := &connectOperation{}
	var filePaths []string
	ctx, out := operationContext(t, op, []string{testID}, func(r *http.Request) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/paths") {
			return http.StatusOK, strings.Replace(entryJSON, `"type":"file"`, `"type":"directory"`, 1)
		}
		assert.Equal(t, contentPrefix+"/files", r.URL.Path)
		value := r.URL.Query().Get("path")
		filePaths = append(filePaths, value)
		switch value {
		case "dir0/file.txt":
			return http.StatusOK, "nested file\n"
		case "file.txt":
			return http.StatusOK, "root file\n"
		default:
			t.Fatalf("unexpected file path %q", value)
			return 0, ""
		}
	})
	errs := new(bytes.Buffer)
	ctx.Cmd.SetErr(errs)
	ctx.Cmd.SetIn(strings.NewReader("cd /dir0\ncat file.txt\ncat /file.txt\nexit\n"))
	require.NoError(t, op.Run(ctx))
	assert.Empty(t, errs.String())
	assert.Equal(t, []string{"dir0/file.txt", "file.txt"}, filePaths)
	assert.Equal(t, "/ > /dir0 > nested file\n/dir0 > root file\n/dir0 > ", out.String())
}

func TestConnectBinaryCatKeepsSession(t *testing.T) {
	for _, data := range []string{"\x7fELF\x00\xff", "%PDF-1.4\n", "invalid UTF-8: \xff"} {
		op := &connectOperation{}
		var filePaths []string
		ctx, out := operationContext(t, op, []string{testID}, func(r *http.Request) (int, string) {
			switch {
			case strings.HasSuffix(r.URL.Path, "/paths"):
				return http.StatusOK, strings.Replace(entryJSON, `"type":"file"`, `"type":"directory"`, 1)
			case strings.HasSuffix(r.URL.Path, "/directories"):
				assert.Equal(t, "dir0", r.URL.Query().Get("path"))
				return http.StatusOK, `[{"name":"still-here.txt"}]`
			default:
				assert.Equal(t, contentPrefix+"/files", r.URL.Path)
				value := r.URL.Query().Get("path")
				filePaths = append(filePaths, value)
				if value == "dir0/binary" {
					return http.StatusOK, data
				}
				return http.StatusOK, "hello 世界\n"
			}
		})
		errs := new(bytes.Buffer)
		ctx.Cmd.SetErr(errs)
		ctx.Cmd.SetIn(strings.NewReader("cd dir0\ncat binary\nls\ncat text.txt\nexit\n"))
		require.NoError(t, op.Run(ctx))
		assert.Contains(t, errs.String(), "binary or non-UTF-8 content and cannot be displayed")
		assert.Contains(t, errs.String(), "vol ct download <volume-id> <path> -f <local-file>")
		assert.Equal(t, []string{"dir0/binary", "dir0/text.txt"}, filePaths)
		assert.Equal(t, "/ > /dir0 > /dir0 > still-here.txt\n/dir0 > hello 世界\n/dir0 > ", out.String())
		assert.NotContains(t, out.String(), data)
	}
}

func TestConnectEOF(t *testing.T) {
	op := &connectOperation{}
	calls := 0
	ctx, out := operationContext(t, op, []string{testID}, func(*http.Request) (int, string) {
		calls++
		if calls == 1 {
			return http.StatusOK, strings.Replace(entryJSON, `"type":"file"`, `"type":"directory"`, 1)
		}
		return http.StatusOK, "[" + entryJSON + "]"
	})
	ctx.Cmd.SetIn(strings.NewReader("\nll"))
	require.NoError(t, op.Run(ctx))
	assert.Equal(t, 2, calls)
	assert.Contains(t, out.String(), "-rw-r--r--")
	assert.Contains(t, out.String(), "5 B")
	assert.Contains(t, out.String(), "a.txt")
}

func TestConnectValidatesVolumeBeforePrompting(t *testing.T) {
	op := &connectOperation{}
	ctx, out := operationContext(t, op, []string{testID}, func(*http.Request) (int, string) {
		return http.StatusForbidden, `{"code":403,"message":"access denied"}`
	})
	ctx.Cmd.SetIn(strings.NewReader("exit\n"))
	require.ErrorContains(t, op.Run(ctx), "access denied")
	assert.Empty(t, out.String())
}

func TestConnectInvalidVolumeID(t *testing.T) {
	op := &connectOperation{}
	ctx, _ := operationContext(t, op, []string{"invalid"}, func(*http.Request) (int, string) {
		t.Fatal("invalid volume ID must not send requests")
		return 0, ""
	})
	require.ErrorContains(t, op.Run(ctx), "expected UUID")
}

func TestSessionPaths(t *testing.T) {
	s := contentSession{cwd: "/dir0/nested"}
	for _, tt := range []struct{ value, want string }{
		{".", "dir0/nested"}, {"..", "dir0"}, {"../file", "dir0/file"},
		{"/", "."}, {"/file", "file"}, {"../../../file", "file"}, {"/../../file", "file"},
		{"./my file", "dir0/nested/my file"},
	} {
		got, err := s.resolve(tt.value)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got, tt.value)
	}
	for _, value := range []string{"", "a\x00b"} {
		_, err := s.resolve(value)
		require.Error(t, err)
	}
}

func TestSplitCommand(t *testing.T) {
	for _, tt := range []struct {
		line string
		want []string
	}{
		{"  ", nil},
		{"ls", []string{"ls"}},
		{`cat "my file"`, []string{"cat", "my file"}},
		{`cat 'my file'`, []string{"cat", "my file"}},
		{`cat my\ file`, []string{"cat", "my file"}},
		{`cat ""`, []string{"cat", ""}},
		{`cat a"b c"d`, []string{"cat", "ab cd"}},
		{`cat 'a\b'`, []string{"cat", `a\b`}},
		{`cat "$HOME"`, []string{"cat", "$HOME"}},
	} {
		got, err := splitCommand(tt.line)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got, tt.line)
	}
	for _, line := range []string{`cat "unfinished`, `cat 'unfinished`, `cat trailing\`} {
		_, err := splitCommand(line)
		require.Error(t, err)
	}
}

func TestConnectInputAndOutputErrors(t *testing.T) {
	op := &connectOperation{}
	ctx, _ := operationContext(t, op, []string{testID}, func(*http.Request) (int, string) {
		return http.StatusOK, strings.Replace(entryJSON, `"type":"file"`, `"type":"directory"`, 1)
	})
	ctx.Cmd.SetIn(io.LimitReader(strings.NewReader("ls"), 2))
	ctx.Cmd.SetOut(brokenWriter{})
	require.ErrorIs(t, op.Run(ctx), io.ErrClosedPipe)
}
