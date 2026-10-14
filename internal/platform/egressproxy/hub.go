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

// Hub gives every workspace its own socket, so a sandbox can only reach the proxy
// through the folder it was handed and each person's traffic stays apart.
type Hub struct {
	proxy   *Proxy
	servers map[string]net.Listener
	base    string
	mu      sync.Mutex
}

// NewHub keeps one socket folder per workspace under base.
func NewHub(base string, proxy *Proxy) *Hub {
	return &Hub{proxy: proxy, base: base, servers: map[string]net.Listener{}}
}

// Dir returns the folder holding the workspace's socket, serving it on first use.
func (h *Hub) Dir(ctx context.Context, id string) (string, error) {
	sum := sha256.Sum256([]byte(id))
	dir := filepath.Join(h.base, hex.EncodeToString(sum[:nameBytes]))

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.servers[id]; ok {
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

	h.servers[id] = l

	go func() { _ = h.proxy.Serve(l) }()

	return dir, nil
}

// Remove stops serving a workspace and deletes its socket folder.
func (h *Hub) Remove(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if l, ok := h.servers[id]; ok {
		_ = l.Close()

		delete(h.servers, id)
	}

	sum := sha256.Sum256([]byte(id))
	_ = os.RemoveAll(filepath.Join(h.base, hex.EncodeToString(sum[:nameBytes])))
}

// Close stops every socket.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, l := range h.servers {
		_ = l.Close()

		delete(h.servers, id)
	}
}
