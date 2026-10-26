package egressproxy

import (
	"io"
	"sync"
	"time"
)

// Lease is one open tunnel that depends on a grant. Whatever it holds is closed when the grant ends.
type Lease struct {
	ends    time.Time
	grants  *Grants
	entry   *grant
	closers []io.Closer
	mu      sync.Mutex
	done    bool
}

// ID is the grant the tunnel lives under.
func (l *Lease) ID() string { return l.entry.id }

// Ends is when the grant stops counting.
func (l *Lease) Ends() time.Time { return l.ends }

// Add ties a connection to the lease. If the lease is already closed, the connection is closed at once.
func (l *Lease) Add(c io.Closer) {
	l.mu.Lock()

	if l.done {
		l.mu.Unlock()

		_ = c.Close()

		return
	}

	l.closers = append(l.closers, c)
	l.mu.Unlock()
}

// Close closes every connection of the lease and frees its slot. It is safe to call more than once.
func (l *Lease) Close() {
	l.mu.Lock()

	if l.done {
		l.mu.Unlock()

		return
	}

	l.done = true
	closers := l.closers
	l.closers = nil
	l.mu.Unlock()

	for _, c := range closers {
		_ = c.Close()
	}

	l.grants.release(l)
}

// Lease checks that the scope has a live grant for host and, in the same locked step, counts a new tunnel against
// the limits and registers it under the grant. A revoke therefore either sees the tunnel and closes it, or the
// tunnel is never opened. host must be canonical.
func (g *Grants) Lease(scope Scope, host string) (*Lease, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	entry := g.byScope[scope][host]
	if g.closed || !scope.valid() || entry == nil || !g.now().Before(entry.ends) {
		return nil, ErrNotGranted
	}

	if g.open[scope] >= g.limits.PerScope || g.total >= g.limits.Total {
		return nil, ErrTooMany
	}

	lease := &Lease{grants: g, entry: entry, ends: entry.ends}
	entry.live[lease] = struct{}{}
	g.open[scope]++
	g.total++

	return lease, nil
}

func (g *Grants) release(l *Lease) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if l.entry.live != nil {
		delete(l.entry.live, l)
	}

	if g.open[l.entry.scope]--; g.open[l.entry.scope] <= 0 {
		delete(g.open, l.entry.scope)
	}

	g.total--
}
