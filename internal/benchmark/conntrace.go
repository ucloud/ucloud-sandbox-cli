package benchmark

import (
	"context"
	"net/http/httptrace"
	"sync"
	"time"
)

// connTrace measures the time HTTP requests spend obtaining a connection: DNS
// lookup, dialing and the TLS handshake for a new connection, close to zero
// for a reused one.
//
// It pairs every GetConn with the GotConn that follows, so it assumes the
// requests it watches run one after another, as they do within one phase.
type connTrace struct {
	mu      sync.Mutex
	pending time.Time
	total   time.Duration
}

// withConnTrace returns a context whose HTTP requests report their connection
// setup time to the returned trace.
func withConnTrace(ctx context.Context) (context.Context, *connTrace) {
	t := &connTrace{}
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GetConn: func(string) { t.getConn() },
		GotConn: func(httptrace.GotConnInfo) { t.gotConn() },
	}), t
}

func (t *connTrace) getConn() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pending = time.Now()
}

func (t *connTrace) gotConn() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pending.IsZero() {
		return
	}
	t.total += time.Since(t.pending)
	t.pending = time.Time{}
}

// Total returns the connection setup time recorded so far.
func (t *connTrace) Total() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.total
}
