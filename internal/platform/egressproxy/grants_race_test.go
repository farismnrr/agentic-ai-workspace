package egressproxy

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type closer struct{ closed atomic.Bool }

func (c *closer) Close() error { c.closed.Store(true); return nil }

// Many tunnels open while the grant is revoked from another goroutine. Whatever the interleaving, a tunnel that got
// its lease is closed once Revoke returns, and none opens after.
func TestRevokeRacingNewTunnelsLeavesNoneOpen(t *testing.T) {
	const unlimited = 1 << 20

	limits := Limits{PerScope: unlimited, Total: unlimited, Handshake: time.Second, Idle: time.Second, Dial: time.Second}
	g := NewGrants(limits, nil)
	t.Cleanup(g.Close)

	_ = g.Allow("g1", scopeA, goodHost, time.Now().Add(time.Minute))

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		held    []*closer
		stop    atomic.Bool
		started = make(chan struct{})
	)

	for range 16 {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-started

			for !stop.Load() {
				lease, err := g.Lease(scopeA, goodHost)
				if err != nil {
					return
				}

				c := &closer{}
				lease.Add(c)

				mu.Lock()
				held = append(held, c)
				mu.Unlock()
			}
		}()
	}

	close(started)
	time.Sleep(20 * time.Millisecond)
	g.Revoke("g1")
	stop.Store(true)
	wg.Wait()

	for i, c := range held {
		if !c.closed.Load() {
			t.Fatalf("connection %d of %d survived the revoke", i, len(held))
		}
	}

	if g.openCount() != 0 {
		t.Fatalf("%d slots still held", g.openCount())
	}
}
