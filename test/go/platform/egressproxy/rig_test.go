package egressproxy_test

import (
	"bytes"
	"context"
	"net"
	"net/netip"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
)

var (
	publicIP = netip.MustParseAddr("93.184.216.34")
	scopeA   = egressproxy.Scope{Project: "p1", Task: "t1"}
)

// rig is a task-socket proxy whose network is replaced: names resolve to answers, and every connection to an
// "address" really goes to one local echo server that records what it was sent.
type rig struct {
	upstream net.Listener
	t        *testing.T
	grants   *egressproxy.Grants
	hub      *egressproxy.Hub
	answers  func(call int) []netip.Addr
	dialed   chan netip.Addr
	received *sink
	lookups  atomic.Int32
}

type sink struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (r *sink) write(p []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.buf.Write(p)
}

func newRig(t *testing.T, limits egressproxy.Limits, verify egressproxy.Verifier) *rig {
	t.Helper()

	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	r := &rig{t: t, upstream: l, received: &sink{}, dialed: make(chan netip.Addr, 64)}
	r.answers = func(int) []netip.Addr { return []netip.Addr{publicIP} }

	go r.echo()

	proxy := fakeDial([]string{"pypi.org"}, egressproxy.WithLookup(func(context.Context, string) ([]netip.Addr, error) {
		return r.answers(int(r.lookups.Add(1))), nil
	}), egressproxy.WithDialIP(func(ctx context.Context, ip netip.Addr) (net.Conn, error) {
		r.dialed <- ip

		return (&net.Dialer{}).DialContext(ctx, "tcp", l.Addr().String())
	}))

	base, err := os.MkdirTemp("", "eg")
	if err != nil {
		t.Fatal(err)
	}

	r.grants = egressproxy.NewGrants(limits, verify)
	r.hub = egressproxy.NewHub(base, proxy).WithGrants(r.grants)

	t.Cleanup(func() {
		r.grants.Close()
		r.hub.Close()
		_ = l.Close()
		_ = os.RemoveAll(base)
	})

	return r
}

func (r *rig) echo() {
	for {
		c, err := r.upstream.Accept()
		if err != nil {
			return
		}

		go r.serveEcho(c)
	}
}

func (r *rig) serveEcho(c net.Conn) {
	defer func() { _ = c.Close() }()

	buf := make([]byte, 4096)

	for {
		n, err := c.Read(buf)
		if n > 0 {
			r.received.write(buf[:n])
			_, _ = c.Write(buf[:n])
		}

		if err != nil {
			return
		}
	}
}

func (r *rig) allow(id string, scope egressproxy.Scope, life time.Duration) {
	r.t.Helper()

	if err := r.grants.Allow(id, scope, goodHost, time.Now().Add(life)); err != nil {
		r.t.Fatal(err)
	}
}
