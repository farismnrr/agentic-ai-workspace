package egressproxy

import (
	"context"
	"net"
	"net/http"
	"path/filepath"
	"testing"
)

func fakeDial(proxy *Proxy) *Proxy {
	proxy.dial = func(context.Context, string) (net.Conn, error) {
		client, server := net.Pipe()

		go func() { _ = server.Close() }()

		return client, nil
	}

	return proxy
}

func TestExtraHostsReachOnlyTheirOwnSocket(t *testing.T) {
	hub := NewHub(t.TempDir(), fakeDial(New([]string{"pypi.org"})))
	t.Cleanup(hub.Close)

	plain, err := hub.Dir(context.Background(), "ws-a")
	connected, errB := hub.Dir(context.Background(), "ws-a", "github.com")
	other, errC := hub.Dir(context.Background(), "ws-b")

	if err != nil || errB != nil || errC != nil {
		t.Fatal(err, errB, errC)
	}

	if plain == connected {
		t.Fatal("a workspace with extra hosts must not share the socket of one without")
	}

	cases := []struct {
		dir, target string
		want        int
	}{
		{connected, "github.com:443", http.StatusOK},
		{connected, "pypi.org:443", http.StatusOK},
		{plain, "github.com:443", http.StatusForbidden},
		{other, "github.com:443", http.StatusForbidden},
	}

	for _, c := range cases {
		if got := connect(t, filepath.Join(c.dir, socketName), c.target); got != c.want {
			t.Errorf("%s via %s: %d, want %d", c.target, filepath.Base(c.dir), got, c.want)
		}
	}

	hub.Remove("ws-a")

	if _, err = hub.Dir(context.Background(), "ws-a"); err != nil {
		t.Fatalf("workspace should be servable again after Remove: %v", err)
	}
}
