package content

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
	contentapi "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/content/api"
)

const testID = "c8fd09b4-a581-4b69-b087-97b97a49ec4f"
const entryJSON = `{"name":"a.txt","type":"file","path":"dir/a.txt","size":5,"mode":420,"uid":1000,"gid":1001,"accessedTime":"2026-11-20T01:02:03Z","modifiedTime":"2026-11-20T01:02:03Z"}`
const contentPrefix = "/content/" + testID

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func operationContext(t *testing.T, op cmd.Operation, args []string, handler func(*http.Request) (int, string)) (cmd.OperationContext, *bytes.Buffer) {
	t.Helper()
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))
	require.NoError(t, c.Args(c, c.Flags().Args()))
	out := new(bytes.Buffer)
	c.SetOut(out)
	cli, err := client.New(client.Options{
		APIKey: "test-key", APIURL: "https://api.example.com/gateway", VolumeContentURL: "https://files.example.com/content",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, "files.example.com", r.URL.Host)
			assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))
			status, body := handler(r)
			return &http.Response{
				StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(body)), Request: r,
			}, nil
		})},
	})
	require.NoError(t, err)
	return cmd.OperationContext{Context: context.Background(), Cmd: c, Args: c.Flags().Args(), Client: cli}, out
}

func TestUpload(t *testing.T) {
	data := []byte{0, 1, 255, '\n', 'x'}
	for _, fromFile := range []bool{false, true} {
		t.Run(map[bool]string{false: "stdin", true: "file"}[fromFile], func(t *testing.T) {
			op := &uploadOperation{}
			args := []string{testID, "dir/a b?#.txt"}
			if fromFile {
				path := filepath.Join(t.TempDir(), "input.bin")
				require.NoError(t, os.WriteFile(path, data, 0600))
				args = append(args, "-f", path)
			}
			called := false
			ctx, out := operationContext(t, op, args, func(r *http.Request) (int, string) {
				called = true
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, contentPrefix+"/files", r.URL.Path)
				assert.Equal(t, args[1], r.URL.Query().Get("path"))
				assert.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, data, body)
				return http.StatusOK, entryJSON
			})
			ctx.Cmd.SetIn(bytes.NewReader(data))
			require.NoError(t, op.Run(ctx))
			assert.True(t, called)
			assert.Empty(t, out.String())
		})
	}
}

func TestDownload(t *testing.T) {
	data := string([]byte{0, 1, 255, '\n', 'x'})
	for _, toFile := range []bool{false, true} {
		t.Run(map[bool]string{false: "stdout", true: "file"}[toFile], func(t *testing.T) {
			op := &downloadOperation{}
			args := []string{testID, "dir/a b.txt"}
			path := filepath.Join(t.TempDir(), "output.bin")
			if toFile {
				require.NoError(t, os.WriteFile(path, []byte("long old contents"), 0600))
				args = append(args, "--file", path)
			}
			called := false
			ctx, out := operationContext(t, op, args, func(r *http.Request) (int, string) {
				called = true
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, contentPrefix+"/files", r.URL.Path)
				assert.Equal(t, args[1], r.URL.Query().Get("path"))
				return http.StatusOK, data
			})
			require.NoError(t, op.Run(ctx))
			assert.True(t, called)
			if toFile {
				body, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, data, string(body))
				assert.Empty(t, out.String())
			} else {
				assert.Equal(t, data, out.String())
			}
		})
	}
}

func TestDownloadFailurePreservesLocalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0600))
	op := &downloadOperation{}
	ctx, out := operationContext(t, op, []string{testID, "missing", "-f", path}, func(*http.Request) (int, string) {
		return http.StatusNotFound, `{"code":404,"message":"missing file"}`
	})
	require.ErrorIs(t, op.Run(ctx), errdefs.ErrNotFound)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "original", string(body))
	assert.Empty(t, out.String())
}

func TestList(t *testing.T) {
	op := &listOperation{}
	ctx, out := operationContext(t, op, []string{testID, "."}, func(r *http.Request) (int, string) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, contentPrefix+"/directories", r.URL.Path)
		assert.Equal(t, ".", r.URL.Query().Get("path"))
		assert.Equal(t, "1", r.URL.Query().Get("depth"))
		directory := strings.NewReplacer(`"name":"a.txt"`, `"name":"z-dir"`, `"type":"file"`, `"type":"directory"`, `"mode":420`, `"mode":493`).Replace(entryJSON)
		link := strings.Replace(entryJSON, `"type":"file"`, `"type":"symlink","symlinkTarget":"a.txt"`, 1)
		link = strings.Replace(link, `"name":"a.txt"`, `"name":"b-link"`, 1)
		return http.StatusOK, "[" + directory + "," + link + "," + entryJSON + "]"
	})
	require.NoError(t, op.Run(ctx))
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, []string{"-rw-r--r--", "1000", "1001", "5", "Nov", "20", "01:02", "a.txt"}, strings.Fields(lines[0]))
	assert.True(t, strings.HasPrefix(lines[1], "lrw-r--r--"))
	assert.True(t, strings.HasSuffix(lines[1], "b-link -> a.txt"))
	assert.True(t, strings.HasPrefix(lines[2], "drwxr-xr-x"))
	assert.True(t, strings.HasSuffix(lines[2], "z-dir"))
}

func TestEmptyDirectory(t *testing.T) {
	op := &listOperation{}
	ctx, out := operationContext(t, op, []string{testID, "."}, func(*http.Request) (int, string) {
		return http.StatusOK, "[]"
	})
	require.NoError(t, op.Run(ctx))
	assert.Empty(t, out.String())
}

func TestStat(t *testing.T) {
	op := &statOperation{}
	ctx, out := operationContext(t, op, []string{testID, "dir/a.txt"}, func(r *http.Request) (int, string) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, contentPrefix+"/paths", r.URL.Path)
		assert.Equal(t, "dir/a.txt", r.URL.Query().Get("path"))
		return http.StatusOK, entryJSON
	})
	require.NoError(t, op.Run(ctx))
	var entry contentapi.EntryInfo
	require.NoError(t, json.Unmarshal(out.Bytes(), &entry))
	assert.Equal(t, "dir/a.txt", entry.Path)
	assert.Equal(t, int64(0o644), entry.Mode)
}

func TestRemove(t *testing.T) {
	op := &removeOperation{}
	ctx, out := operationContext(t, op, []string{testID, "dir"}, func(r *http.Request) (int, string) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, contentPrefix+"/paths", r.URL.Path)
		assert.Equal(t, "dir", r.URL.Query().Get("path"))
		return http.StatusNoContent, ""
	})
	require.NoError(t, op.Run(ctx))
	assert.Empty(t, out.String())
}

func TestMkdir(t *testing.T) {
	op := &mkdirOperation{}
	called := false
	ctx, out := operationContext(t, op, []string{testID, "new dir"}, func(r *http.Request) (int, string) {
		called = true
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, contentPrefix+"/directories", r.URL.Path)
		assert.Empty(t, r.URL.RawQuery)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{"path": "new dir"}, body)
		return http.StatusOK, strings.Replace(entryJSON, `"type":"file"`, `"type":"directory"`, 1)
	})
	require.NoError(t, op.Run(ctx))
	assert.True(t, called)
	assert.Empty(t, out.String())
}

func operations() []cmd.Operation {
	return []cmd.Operation{&uploadOperation{}, &downloadOperation{}, &listOperation{}, &mkdirOperation{}, &removeOperation{}, &statOperation{}}
}

func TestInvalidArgumentsDoNotSendRequests(t *testing.T) {
	for _, op := range operations() {
		for _, args := range [][]string{{"invalid", "."}, {testID, ""}, {testID, "/"}, {testID, "../secret"}, {testID, "dir/../secret"}, {testID, "a\x00b"}} {
			ctx, _ := operationContext(t, op, args, func(*http.Request) (int, string) {
				t.Fatal("invalid arguments must not send requests")
				return 0, ""
			})
			require.Error(t, op.Run(ctx))
		}
	}
}

func TestAPIErrors(t *testing.T) {
	for _, op := range operations() {
		t.Run(op.Command().Name(), func(t *testing.T) {
			ctx, out := operationContext(t, op, []string{testID, "missing"}, func(*http.Request) (int, string) {
				return http.StatusForbidden, `{"code":403,"message":"access denied"}`
			})
			ctx.Cmd.SetIn(strings.NewReader("data"))
			err := op.Run(ctx)
			require.ErrorIs(t, err, errdefs.ErrForbidden)
			assert.Contains(t, err.Error(), "access denied")
			assert.Empty(t, out.String())
		})
	}
}

func TestMissingUploadFile(t *testing.T) {
	op := &uploadOperation{}
	ctx, _ := operationContext(t, op, []string{testID, "file", "-f", filepath.Join(t.TempDir(), "missing")}, func(*http.Request) (int, string) {
		t.Fatal("missing local file must not send a request")
		return 0, ""
	})
	require.ErrorIs(t, op.Run(ctx), os.ErrNotExist)
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestOutputErrors(t *testing.T) {
	for _, op := range []cmd.Operation{&downloadOperation{}, &listOperation{}, &statOperation{}} {
		ctx, _ := operationContext(t, op, []string{testID, "file"}, func(*http.Request) (int, string) {
			if _, ok := op.(*listOperation); ok {
				return http.StatusOK, "[" + entryJSON + "]"
			}
			return http.StatusOK, entryJSON
		})
		ctx.Cmd.SetOut(brokenWriter{})
		require.ErrorIs(t, op.Run(ctx), io.ErrClosedPipe)
	}
}

func TestCommandArguments(t *testing.T) {
	for _, op := range operations() {
		c := op.Command()
		for _, args := range [][]string{nil, {testID}, {testID, ".", "extra"}} {
			assert.Error(t, c.Args(c, args), c.Name())
		}
		assert.NoError(t, c.Args(c, []string{testID, "."}), c.Name())
	}
}

func TestUploadForce(t *testing.T) {
	op := &uploadOperation{}
	ctx, out := operationContext(t, op, []string{testID, "nested/file", "--force"}, func(r *http.Request) (int, string) {
		assert.Equal(t, "true", r.URL.Query().Get("force"))
		return http.StatusOK, entryJSON
	})
	ctx.Cmd.SetIn(strings.NewReader("hello"))
	require.NoError(t, op.Run(ctx))
	assert.Empty(t, out.String())
}

func TestMkdirParents(t *testing.T) {
	op := &mkdirOperation{}
	ctx, _ := operationContext(t, op, []string{testID, "nested/dir", "-p"}, func(r *http.Request) (int, string) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{"path": "nested/dir", "createParents": true}, body)
		return http.StatusOK, entryJSON
	})
	require.NoError(t, op.Run(ctx))
}

func TestRename(t *testing.T) {
	op := &renameOperation{}
	ctx, out := operationContext(t, op, []string{testID, "dir/a.txt", "dir/b.txt"}, func(r *http.Request) (int, string) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, contentPrefix+"/paths/rename", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{"sourcePath": "dir/a.txt", "destinationPath": "dir/b.txt"}, body)
		return http.StatusOK, entryJSON
	})
	require.NoError(t, op.Run(ctx))
	assert.Empty(t, out.String())
}

func TestRenameRejectsInvalidDestination(t *testing.T) {
	op := &renameOperation{}
	for _, destination := range []string{"", "/outside", "dir/../outside"} {
		ctx, _ := operationContext(t, op, []string{testID, "a.txt", destination}, func(*http.Request) (int, string) {
			t.Fatal("invalid destination must not send requests")
			return 0, ""
		})
		require.Error(t, op.Run(ctx))
	}
}
