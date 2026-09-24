// Package logs walks a log listing whose pagination cursor is a millisecond
// timestamp.
//
// Both the sandbox log endpoint and the template build log endpoint page that
// way, and both need the same care: asking for entries from millisecond N
// returns every entry of millisecond N again, so a page overlaps the one
// before it and the repeats have to be recognised by their content.
package logs

import "time"

// Cursor tracks the forward pagination position of a log listing.
//
// It is not safe for concurrent use; one listing is walked by one goroutine.
type Cursor[T any] struct {
	ms   int64
	seen map[string]struct{}

	timestamp func(T) time.Time
	signature func(T) string
}

// NewCursor returns a cursor over entries of type T.
//
// timestamp reads an entry's time, which is what the endpoint pages on.
// signature identifies an entry by its content, so the same entry arriving
// twice across two pages is reported once; two entries of the same millisecond
// that differ in any way must produce different signatures.
func NewCursor[T any](timestamp func(T) time.Time, signature func(T) string) *Cursor[T] {
	return &Cursor[T]{
		seen:      map[string]struct{}{},
		timestamp: timestamp,
		signature: signature,
	}
}

// At returns the millisecond the next page should be requested from. Zero
// means the listing has not moved yet, so the request should carry no cursor.
func (c *Cursor[T]) At() int64 { return c.ms }

// SeekTo starts the walk at a millisecond the caller chose.
func (c *Cursor[T]) SeekTo(ms int64) {
	c.ms = ms
	c.seen = map[string]struct{}{}
}

// Advance takes a page and returns the entries of it that have not been
// reported yet, along with whether another page should be requested right
// away. pageFull tells whether the page reached the requested limit, meaning
// more entries may be waiting.
func (c *Cursor[T]) Advance(page []T, pageFull bool) ([]T, bool) {
	if len(page) == 0 {
		return nil, false
	}

	fresh := make([]T, 0, len(page))
	for _, entry := range page {
		ms := c.timestamp(entry).UnixMilli()
		if ms < c.ms {
			continue
		}
		if ms == c.ms {
			if _, ok := c.seen[c.signature(entry)]; ok {
				continue
			}
		}
		fresh = append(fresh, entry)
	}

	lastMs := c.timestamp(page[len(page)-1]).UnixMilli()
	if pageFull && lastMs == c.ms && len(fresh) == 0 {
		// A full page holds nothing but already reported entries of the cursor
		// millisecond. Step over it, otherwise the same page repeats forever.
		c.ms++
		c.seen = map[string]struct{}{}
		return fresh, true
	}

	if lastMs != c.ms {
		c.ms = lastMs
		c.seen = map[string]struct{}{}
	}
	for _, entry := range page {
		if c.timestamp(entry).UnixMilli() == c.ms {
			c.seen[c.signature(entry)] = struct{}{}
		}
	}

	return fresh, pageFull
}
