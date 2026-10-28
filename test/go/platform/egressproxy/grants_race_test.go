package egressproxy_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
)

type closer struct{ closed atomic.Bool }

func (c *closer) Close() error { c.closed.Store(true); return nil }

// Many tunnels open while the grant is revoked from another goroutine. Whatever the interleaving, a tunnel that got
// its lease is closed once Revoke returns, and none opens after.
func TestRevokeRacingNewTunnelsLeavesNoneOpen(t *testing.T) {
	const unlimited = 1 << 20

	limits := egressproxy.Limits{
		PerScope: unlimited, Total: unlimited, Handshake: time.Second, Idle: time.Second,
		Dial: time.Second,
	}
	g := egressproxy.NewGrants(limits, nil)
	t.Cleanup(g.Close)

	_ = g.Allow("g1", scopeA, goodHost, time.Now().Add(time.Minute))

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		held  []*closer
		stop  atomic.Bool
		ready = make(chan bool, 16)
	)

	for range 16 {
		wg.Add(1)

		go func() {
			defer wg.Done()
			leaseUntilRevoked(g, &stop, &mu, &held, ready)
		}()
	}

	for range 16 {
		if !<-ready {
			t.Fatal("a lease worker exited before acquiring a tunnel")
		}
	}
	g.Revoke("g1")
	stop.Store(true)
	wg.Wait()

	for i, c := range held {
		if !c.closed.Load() {
			t.Fatalf("connection %d of %d survived the revoke", i, len(held))
		}
	}

	if g.OpenTunnels() != 0 {
		t.Fatalf("%d slots still held", g.OpenTunnels())
	}
}

func leaseUntilRevoked(
	g *egressproxy.Grants, stop *atomic.Bool, mu *sync.Mutex, held *[]*closer, ready chan<- bool,
) {
	firstLease := true
	for !stop.Load() {
		lease, err := g.Lease(scopeA, goodHost)
		if err != nil {
			if firstLease {
				ready <- false
			}
			return
		}

		connection := &closer{}
		lease.Add(connection)
		mu.Lock()
		*held = append(*held, connection)
		mu.Unlock()
		if firstLease {
			ready <- true
			firstLease = false
		}
	}
}
