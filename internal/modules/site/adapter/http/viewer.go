package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	viewSocketName = "view.sock"
	viewSocketMode = 0o666
	viewDirMode    = 0o711
	maxViewBody    = 1 << 20
	viewHeaderWait = 5 * time.Second
)

var errNotInView = errors.New("the preview serves only the app's own address")

// Viewer lets the AI's browser open the app through an isolated, temporary socket for one capture.
type Viewer struct{}

// NewViewer makes the viewer.
func NewViewer() Viewer { return Viewer{} }

// Open listens in a private per-capture directory and returns its socket and cleanup function.
func (Viewer) Open(
	ctx context.Context, appSocket, base, prefix string, jar http.CookieJar,
) (string, func(), error) {
	dir, err := os.MkdirTemp(filepath.Dir(filepath.Dir(appSocket)), "v-")
	if err != nil {
		return "", nil, fmt.Errorf("open the preview: %w", err)
	}
	if err = os.Chmod(dir, viewDirMode); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, fmt.Errorf("open the preview: %w", err)
	}
	path := filepath.Join(dir, viewSocketName)

	listener, err := (&net.ListenConfig{}).Listen(ctx, "unix", path)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, fmt.Errorf("open the preview: %w", err)
	}

	// The browser's container runs as another user and must be able to connect.
	if err = os.Chmod(path, viewSocketMode); err != nil {
		_ = listener.Close()
		_ = os.RemoveAll(dir)

		return "", nil, fmt.Errorf("open the preview: %w", err)
	}

	server := &http.Server{
		Handler:           viewHandler{socket: appSocket, base: base, prefix: prefix, jar: jar},
		ReadHeaderTimeout: viewHeaderWait,
	}

	go func() { _ = server.Serve(listener) }()
	done := make(chan struct{})
	var once sync.Once
	closeView := func() {
		once.Do(func() {
			close(done)
			_ = server.Close()
			_ = os.RemoveAll(dir)
		})
	}
	go func() {
		select {
		case <-ctx.Done():
			closeView()
		case <-done:
		}
	}()

	return path, closeView, nil
}

type viewHandler struct {
	jar    http.CookieJar
	socket string
	base   string
	prefix string
}

func (v viewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, err := v.inside(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	}
	if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
		viewerPreflight(w, r)
		return
	}

	body, _ := io.ReadAll(io.LimitReader(r.Body, maxViewBody))

	out, failure := relay(r.Context(), relayRequest{
		Socket: v.socket, Method: r.Method, Path: rest, Query: r.URL.RawQuery, Body: body, Header: r.Header,
		Base: v.base, Prefix: v.prefix, ClientIP: "127.0.0.1", AgentJar: v.jar,
	})
	if failure != nil {
		http.Error(w, failure.text, failure.status)

		return
	}

	for name, values := range out.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(out.Status)
	_, _ = w.Write(out.Body)
}

func viewerPreflight(w http.ResponseWriter, r *http.Request) {
	secure(w.Header())
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Max-Age", "600")
	if asked := r.Header.Get("Access-Control-Request-Headers"); asked != "" {
		w.Header().Set("Access-Control-Allow-Headers", asked)
	}
	w.WriteHeader(http.StatusNoContent)
}

// inside returns the path the app receives, or an error for a path outside the app's address.
func (v viewHandler) inside(path string) (string, error) {
	if path == v.prefix {
		return "/", nil
	}

	rest, found := strings.CutPrefix(path, v.prefix+"/")
	if !found {
		return "", errNotInView
	}

	return "/" + rest, nil
}
