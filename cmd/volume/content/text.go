package content

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

var errBinaryContent = errors.New("file contains binary or non-UTF-8 content; use -f <local-file> to download it")

func writeText(out io.Writer, body io.Reader) error {
	writer := &textWriter{out: out}
	if _, err := io.Copy(writer, body); err != nil {
		return err
	}
	return writer.Flush()
}

// textWriter validates each chunk before forwarding it to a terminal. It only
// retains the initial MIME sample (at most 511 bytes) or an incomplete UTF-8 rune.
type textWriter struct {
	out     io.Writer
	pending []byte
	sniffed bool
	err     error
}

func (w *textWriter) Write(p []byte) (int, error) {
	return w.write(p, false)
}

// Flush validates the last sample or partial rune without closing the output.
func (w *textWriter) Flush() error {
	_, err := w.write(nil, true)
	return err
}

func (w *textWriter) write(p []byte, final bool) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	defer func() {
		if err != nil {
			w.err = err
		}
	}()
	previous := len(w.pending)
	data := p
	if previous > 0 {
		data = make([]byte, previous+len(p))
		copy(data, w.pending)
		copy(data[previous:], p)
	}
	if !w.sniffed {
		if len(data) < 512 && !final {
			w.pending = append(w.pending[:0], data...)
			return len(p), nil
		}
		if !strings.HasPrefix(http.DetectContentType(data[:min(len(data), 512)]), "text/") {
			return 0, errBinaryContent
		}
		w.sniffed = true
	}

	complete := 0
	for complete < len(data) {
		if !utf8.FullRune(data[complete:]) {
			if final {
				return 0, errBinaryContent
			}
			break
		}
		r, size := utf8.DecodeRune(data[complete:])
		if (r == utf8.RuneError && size == 1) || (unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' && r != '\f') {
			return 0, errBinaryContent
		}
		complete += size
	}

	if complete > 0 {
		written, writeErr := w.out.Write(data[:complete])
		if writeErr == nil && written != complete {
			writeErr = io.ErrShortWrite
		}
		if writeErr != nil {
			return max(0, written-previous), writeErr
		}
	}
	w.pending = append(w.pending[:0], data[complete:]...)
	return len(p), nil
}
