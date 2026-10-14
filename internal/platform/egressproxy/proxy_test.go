package egressproxy

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func connect(t *testing.T, path, target string) int {
	t.Helper()

	conn, err := (&net.Dialer{}).DialContext(context.Background(), "unix", path)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}

	t.Cleanup(func() { _ = conn.Close() })

	if _, err = conn.Write([]byte("CONNECT " + target + " HTTP/1.1\r\nHost: " + target + "\r\n\r\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	res, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	_ = res.Body.Close()

	return res.StatusCode
}

func serve(t *testing.T, proxy *Proxy) string {
	t.Helper()

	hub := NewHub(t.TempDir(), proxy)
	t.Cleanup(hub.Close)

	dir, err := hub.Dir(context.Background(), "workspace-a")
	if err != nil {
		t.Fatalf("hub: %v", err)
	}

	return filepath.Join(dir, socketName)
}

func TestProxyTunnelsOnlyAllowedHostsOnHTTPS(t *testing.T) {
	proxy := New([]string{"pypi.org", "localhost"})
	proxy.dial = func(context.Context, string) (net.Conn, error) {
		client, server := net.Pipe()

		go func() { _ = server.Close() }()

		return client, nil
	}
	path := serve(t, proxy)

	cases := map[string]int{
		"pypi.org:443":        http.StatusOK,
		"PyPI.org:443":        http.StatusOK,
		"example.com:443":     http.StatusForbidden,
		"pypi.org:80":         http.StatusForbidden,
		"169.254.169.254:443": http.StatusForbidden,
	}

	for target, want := range cases {
		if got := connect(t, path, target); got != want {
			t.Errorf("CONNECT %s = %d, want %d", target, got, want)
		}
	}
}

func TestProxyRefusesAllowedNamesThatResolveInside(t *testing.T) {
	path := serve(t, New([]string{"localhost"}))

	if got := connect(t, path, "localhost:443"); got != http.StatusBadGateway {
		t.Fatalf("a name resolving to loopback = %d, want %d", got, http.StatusBadGateway)
	}
}

func TestHubKeepsWorkspacesApartAndRemoves(t *testing.T) {
	hub := NewHub(t.TempDir(), New(nil))
	t.Cleanup(hub.Close)

	a, errA := hub.Dir(context.Background(), "a")
	b, errB := hub.Dir(context.Background(), "b")

	if errA != nil || errB != nil || a == b {
		t.Fatalf("dirs %q %q errors %v %v", a, b, errA, errB)
	}

	hub.Remove("a")

	if _, err := os.Stat(a); !os.IsNotExist(err) {
		t.Fatalf("socket folder should be gone, got %v", err)
	}
}
