package content

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

func TestTerminalWriterRejectsBinaryChunks(t *testing.T) {
	for _, tt := range []struct{ name, data string }{
		{"NUL", "hello\x00world"},
		{"PNG", "\x89PNG\r\n\x1a\n"},
		{"PDF with ASCII contents", "%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\n"},
		{"invalid UTF-8", "hello\xff"},
		{"UTF-16", "\xff\xfeh\x00i\x00"},
		{"terminal escape", "\x1b[31mhello"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			err := writeText(out, iotest.OneByteReader(strings.NewReader(tt.data)))
			require.ErrorContains(t, err, "-f <local-file>")
			assert.Empty(t, out.String())
		})
	}
}

func TestTerminalWriterPreservesTextAcrossChunks(t *testing.T) {
	for _, data := range []string{
		"", "hello\n", "你好 🌍\n", "\ufeffUTF-8 BOM\r\n", "one\ttwo\r\n\f", "valid replacement rune: \ufffd",
		strings.Repeat("a", (32<<10)-1) + "你好🌍\n",
	} {
		out := new(bytes.Buffer)
		require.NoError(t, writeText(out, iotest.OneByteReader(strings.NewReader(data))))
		assert.Equal(t, data, out.String())
	}
}

func TestTerminalWriterStreamsBeforeEOFAndRejectsLaterBinary(t *testing.T) {
	out := new(bytes.Buffer)
	writer := &textWriter{out: out}
	text := strings.Repeat("text\n", 1000)
	n, err := writer.Write([]byte(text))
	require.NoError(t, err)
	assert.Equal(t, len(text), n)
	assert.Equal(t, text, out.String())
	n, err = writer.Write([]byte("more text\x00binary"))
	require.ErrorIs(t, err, errBinaryContent)
	assert.Zero(t, n)
	assert.Equal(t, text, out.String())
	require.ErrorIs(t, writer.Flush(), errBinaryContent)
}

func TestTerminalWriterRejectsIncompleteUTF8AtEOF(t *testing.T) {
	out := new(bytes.Buffer)
	text := strings.Repeat("a", 600)
	require.ErrorIs(t, writeText(out, strings.NewReader(text+"\xe4\xbd")), errBinaryContent)
	assert.Equal(t, text, out.String())
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestTerminalWriterPropagatesReadFailure(t *testing.T) {
	want := errors.New("download interrupted")
	text := strings.Repeat("partial text\n", 100)
	body := io.MultiReader(strings.NewReader(text), failingReader{want})
	out := new(bytes.Buffer)
	require.ErrorIs(t, writeText(out, body), want)
	assert.Equal(t, text, out.String())
}

func TestTerminalWriterPropagatesOutputFailure(t *testing.T) {
	require.ErrorIs(t, writeText(brokenWriter{}, strings.NewReader("text\n")), io.ErrClosedPipe)
}

func TestDownloadBinaryToPipeAndRedirectedFile(t *testing.T) {
	data := "\x00\xff\x1b[31m" + strings.Repeat("binary\x00", 10000)
	for _, redirect := range []bool{false, true} {
		op := &downloadOperation{}
		ctx, out := operationContext(t, op, []string{testID, "file"}, func(*http.Request) (int, string) {
			return http.StatusOK, data
		})
		progress := new(bytes.Buffer)
		ctx.Cmd.SetErr(progress)
		if redirect {
			filePath := filepath.Join(t.TempDir(), "redirected.bin")
			file, err := os.Create(filePath)
			require.NoError(t, err)
			ctx.Cmd.SetOut(file)
			require.NoError(t, op.Run(ctx))
			require.NoError(t, file.Close())
			body, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, data, string(body))
		} else {
			readEnd, writeEnd, err := os.Pipe()
			require.NoError(t, err)
			defer readEnd.Close()
			defer writeEnd.Close()
			ctx.Cmd.SetOut(writeEnd)
			done := make(chan error, 1)
			go func() {
				_, err := io.Copy(out, readEnd)
				done <- err
			}()
			runErr := op.Run(ctx)
			require.NoError(t, writeEnd.Close())
			require.NoError(t, runErr)
			require.NoError(t, <-done)
			assert.Equal(t, data, out.String())
		}
		assert.Empty(t, progress.String())
	}
}

func TestDownloadUnknownLengthAndEmptyFiles(t *testing.T) {
	for _, data := range []string{"", "\x00\xffbinary"} {
		t.Run(map[bool]string{true: "empty", false: "binary"}[data == ""], func(t *testing.T) {
			op := &downloadOperation{}
			filePath := filepath.Join(t.TempDir(), "output.bin")
			ctx, out := operationContext(t, op, []string{testID, "file", "-f", filePath}, func(*http.Request) (int, string) {
				return http.StatusOK, data
			})
			transport := ctx.Client.Transport().HTTPClient()
			original := transport.Transport
			transport.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				response, err := original.RoundTrip(r)
				if response != nil {
					response.ContentLength = -1
				}
				return response, err
			})
			progress := new(bytes.Buffer)
			ctx.Cmd.SetErr(progress)
			require.NoError(t, op.Run(ctx))
			body, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, data, string(body))
			assert.Empty(t, out.String())
			assert.Contains(t, progress.String(), "Downloading")
			assert.NotContains(t, progress.String(), "NaN")
		})
	}
}

func TestUploadEmptyFileWithProgress(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "empty")
	require.NoError(t, os.WriteFile(filePath, nil, 0600))
	op := &uploadOperation{}
	ctx, out := operationContext(t, op, []string{testID, "file", "-f", filePath}, func(r *http.Request) (int, string) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Empty(t, body)
		return http.StatusOK, entryJSON
	})
	progress := new(bytes.Buffer)
	ctx.Cmd.SetErr(progress)
	require.NoError(t, op.Run(ctx))
	assert.Empty(t, out.String())
	assert.Contains(t, progress.String(), "Uploading")
	assert.NotContains(t, progress.String(), "NaN")
}

func TestUploadFailureDoesNotFinishProgress(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "input")
	require.NoError(t, os.WriteFile(filePath, []byte("upload data"), 0600))
	op := &uploadOperation{}
	ctx, out := operationContext(t, op, []string{testID, "file", "-f", filePath}, func(*http.Request) (int, string) {
		return http.StatusForbidden, `{"code":403,"message":"access denied"}`
	})
	progress := new(bytes.Buffer)
	ctx.Cmd.SetErr(progress)
	require.ErrorIs(t, op.Run(ctx), errdefs.ErrForbidden)
	assert.Empty(t, out.String())
	assert.Contains(t, progress.String(), "Uploading")
	assert.NotContains(t, progress.String(), "100%")
}
