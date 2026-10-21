package egressproxy

import (
	"context"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

// recorder replaces both dialers and remembers which one was asked for which address.
type recorder struct {
	public, target []string
	mu             sync.Mutex
}

func (r *recorder) install(p *Proxy) *Proxy {
	pipe := func(into *[]string) dialFunc {
		return func(_ context.Context, address string) (net.Conn, error) {
			r.mu.Lock()
			*into = append(*into, address)
			r.mu.Unlock()

			client, server := net.Pipe()

			go func() { _ = server.Close() }()

			return client, nil
		}
	}
	p.dial, p.dialTarget = pipe(&r.public), pipe(&r.target)

	return p
}

func TestExactTargetReachesOnlyThatHostAndPort(t *testing.T) {
	rec := &recorder{}
	hub := NewHub(t.TempDir(), rec.install(New([]string{"pypi.org"})))
	t.Cleanup(hub.Close)

	withDB, err := hub.Dir(context.Background(), "ws-a", "DB.internal:5432", "github.com")
	plain, errPlain := hub.Dir(context.Background(), "ws-a")

	if err != nil || errPlain != nil {
		t.Fatal(err, errPlain)
	}

	cases := []struct {
		dir, target string
		want        int
	}{
		{withDB, "db.internal:5432", http.StatusOK},
		{withDB, "db.internal:5433", http.StatusForbidden},
		{withDB, "db.internal:443", http.StatusForbidden},
		{withDB, "redis.internal:5432", http.StatusForbidden},
		{withDB, "github.com:443", http.StatusOK},
		{withDB, "github.com:5432", http.StatusForbidden},
		{plain, "db.internal:5432", http.StatusForbidden},
	}

	for _, c := range cases {
		if got := connect(t, filepath.Join(c.dir, socketName), c.target); got != c.want {
			t.Errorf("%s: %d, want %d", c.target, got, c.want)
		}
	}

	if len(rec.target) != 1 || rec.target[0] != "db.internal:5432" {
		t.Errorf("only the exact target uses the target dialer, got %v", rec.target)
	}

	if len(rec.public) != 1 || rec.public[0] != "github.com:443" {
		t.Errorf("HTTPS hosts keep the public-only dialer, got %v", rec.public)
	}
}

func TestExactTargetMayBeOnThisComputerButNeverTheServerItself(t *testing.T) {
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = l.Close() })

	go func() {
		for {
			conn, acceptErr := l.Accept()
			if acceptErr != nil {
				return
			}

			_ = conn.Close()
		}
	}()

	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port) //nolint:forcetypeassert // a TCP listener has a TCP address
	target := "127.0.0.1:" + port

	open := New(nil).WithOwnPort("1")
	hub := NewHub(t.TempDir(), open)
	t.Cleanup(hub.Close)

	dir, err := hub.Dir(context.Background(), "ws", target)
	if err != nil {
		t.Fatal(err)
	}

	if got := connect(t, filepath.Join(dir, socketName), target); got != http.StatusOK {
		t.Errorf("a database on this computer is reachable when saved: %d", got)
	}

	self := NewHub(t.TempDir(), New(nil).WithOwnPort(port))
	t.Cleanup(self.Close)

	dir, err = self.Dir(context.Background(), "ws", target)
	if err != nil {
		t.Fatal(err)
	}

	if got := connect(t, filepath.Join(dir, socketName), target); got != http.StatusBadGateway {
		t.Errorf("the server's own port must be refused: %d", got)
	}
}
