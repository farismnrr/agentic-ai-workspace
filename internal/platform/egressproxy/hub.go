package egressproxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

const (
	dirMode    = 0o750
	socketMode = 0o660
	socketName = "p.sock"
	nameBytes  = 6
	// maxSocketPath is below the 108 bytes a Unix socket address can hold.
	maxSocketPath = 100
)

var errPathTooLong = errors.New("the data folder path is too long for a Unix socket")

// served is one open socket: the workspace it belongs to and its folder.
type served struct {
	l   net.Listener
	id  string
	dir string
}

// Hub gives every workspace its own socket, so a sandbox can only reach the proxy
// through the folder it was handed and each person's traffic stays apart.
type Hub struct {
	proxy   *Proxy
	servers map[string]served
	base    string
	mu      sync.Mutex
}

// NewHub keeps one socket folder per workspace under base.
func NewHub(base string, proxy *Proxy) *Hub {
	return &Hub{proxy: proxy, base: base, servers: map[string]served{}}
}

// Dir returns the folder holding the workspace's socket, serving it on first use. Extra hosts are
// reachable through that socket only. Each set of hosts has a socket of its own, so a web app
// without them and a command with them can run in the same workspace at once, and one person's
// access never carries over to another workspace.
func (h *Hub) Dir(ctx context.Context, id string, extra ...string) (string, error) {
	key := id + "\x00" + strings.Join(slices.Sorted(slices.Values(extra)), ",")
	sum := sha256.Sum256([]byte(key))
	dir := filepath.Join(h.base, hex.EncodeToString(sum[:nameBytes]))

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.servers[key]; ok {
		return dir, nil
	}

	path := filepath.Join(dir, socketName)
	if len(path) > maxSocketPath {
		return "", fmt.Errorf("%w: %s", errPathTooLong, path)
	}

	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create socket folder: %w", err)
	}

	_ = os.Remove(path)

	l, err := (&net.ListenConfig{}).Listen(ctx, "unix", path)
	if err != nil {
		return "", fmt.Errorf("listen on socket: %w", err)
	}

	if err = os.Chmod(path, socketMode); err != nil {
		_ = l.Close()

		return "", fmt.Errorf("protect socket: %w", err)
	}

	h.servers[key] = served{l: l, id: id, dir: dir}
	proxy := h.proxy.WithHosts(extra)

	go func() { _ = proxy.Serve(l) }()

	return dir, nil
}

// Remove stops serving a workspace and deletes its socket folders.
func (h *Hub) Remove(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for key, s := range h.servers {
		if s.id == id {
			_ = s.l.Close()
			_ = os.RemoveAll(s.dir)

			delete(h.servers, key)
		}
	}
}

// Close stops every socket.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for key, s := range h.servers {
		_ = s.l.Close()

		delete(h.servers, key)
	}
}
