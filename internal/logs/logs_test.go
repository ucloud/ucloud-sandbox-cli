package logs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// entry is a minimal log entry: a millisecond timestamp and a message.
type entry struct {
	ms      int64
	message string
}

func at(ms int64, message string) entry { return entry{ms: ms, message: message} }

func newTestCursor() *Cursor[entry] {
	return NewCursor(
		func(e entry) time.Time { return time.UnixMilli(e.ms).UTC() },
		func(e entry) string { return e.message },
	)
}

func TestCursorAdvance(t *testing.T) {
	cursor := newTestCursor()

	first := []entry{at(1000, "a"), at(2000, "b")}

	fresh, more := cursor.Advance(first, true)
	assert.Equal(t, first, fresh)
	assert.True(t, more, "a full page means another one may be waiting")
	assert.Equal(t, int64(2000), cursor.At())

	// The endpoint replays the entries of the cursor millisecond, which must
	// not be reported twice.
	second := []entry{at(2000, "b"), at(2000, "c"), at(3000, "d")}

	fresh, more = cursor.Advance(second, false)
	assert.Equal(t, []entry{at(2000, "c"), at(3000, "d")}, fresh)
	assert.False(t, more)
	assert.Equal(t, int64(3000), cursor.At())
}

func TestCursorAdvanceOnEmptyPage(t *testing.T) {
	cursor := newTestCursor()

	fresh, more := cursor.Advance(nil, false)
	assert.Empty(t, fresh)
	assert.False(t, more)
	assert.Zero(t, cursor.At())
}

func TestCursorStepsOverARepeatedFullPage(t *testing.T) {
	cursor := newTestCursor()

	page := []entry{at(2000, "a"), at(2000, "b")}

	fresh, more := cursor.Advance(page, true)
	require.Len(t, fresh, 2)
	require.True(t, more)
	require.Equal(t, int64(2000), cursor.At())

	// A full page of nothing but already reported entries: the cursor has to
	// step past the millisecond, otherwise the same page repeats forever.
	fresh, more = cursor.Advance(page, true)
	assert.Empty(t, fresh)
	assert.True(t, more)
	assert.Equal(t, int64(2001), cursor.At())
}

func TestCursorSkipsEntriesBeforeTheCursor(t *testing.T) {
	cursor := newTestCursor()
	cursor.SeekTo(2000)

	// An endpoint that ignores the cursor, or a page that starts earlier than
	// asked, must not replay what is already behind the walk.
	fresh, _ := cursor.Advance([]entry{at(1000, "old"), at(3000, "new")}, false)
	assert.Equal(t, []entry{at(3000, "new")}, fresh)
}

func TestCursorSeekTo(t *testing.T) {
	cursor := newTestCursor()
	assert.Zero(t, cursor.At(), "a fresh cursor asks for no cursor at all")

	cursor.SeekTo(4000)
	assert.Equal(t, int64(4000), cursor.At())

	// Seeking clears what was reported, so the new position starts clean.
	fresh, _ := cursor.Advance([]entry{at(4000, "a")}, false)
	assert.Equal(t, []entry{at(4000, "a")}, fresh)
}

func TestCursorReportsAnEntryPerSignature(t *testing.T) {
	cursor := newTestCursor()

	// Two entries share a millisecond but differ in content, so both are new.
	fresh, _ := cursor.Advance([]entry{at(2000, "a"), at(2000, "b")}, false)
	require.Len(t, fresh, 2)

	// The same two come back: neither is.
	fresh, _ = cursor.Advance([]entry{at(2000, "a"), at(2000, "b")}, false)
	assert.Empty(t, fresh)

	// A third one of that millisecond is.
	fresh, _ = cursor.Advance([]entry{at(2000, "a"), at(2000, "c")}, false)
	assert.Equal(t, []entry{at(2000, "c")}, fresh)
}

func TestCursorAPartialPageEndsTheWalk(t *testing.T) {
	cursor := newTestCursor()

	_, more := cursor.Advance([]entry{at(1000, "a")}, false)
	assert.False(t, more, "a page short of the limit is the last one")
}
