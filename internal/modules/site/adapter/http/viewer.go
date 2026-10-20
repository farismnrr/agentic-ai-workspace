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
	"time"
)

const (
	viewSocketName = "view.sock"
	viewSocketMode = 0o666
	maxViewBody    = 1 << 20
	viewHeaderWait = 5 * time.Second
)

var errNotInView = errors.New("the preview serves only the app's own address")

// Viewer lets the AI's browser open the app the way the person's browser does, without a login. It serves the
// same proxy code (the address, the moved links, the sandbox headers) on a socket that lives only while a picture
// is taken, inside the app's own socket folder, so nobody else can reach it.
type Viewer struct{}

// NewViewer makes the viewer.
func NewViewer() Viewer { return Viewer{} }

// Open listens next to the app's socket and returns the socket's path and the function that closes it. The
// browser asks for pages under prefix, as the person's browser does.
func (Viewer) Open(ctx context.Context, appSocket, base, prefix string) (string, func(), error) {
	path := filepath.Join(filepath.Dir(appSocket), viewSocketName)
	_ = os.Remove(path)

	listener, err := (&net.ListenConfig{}).Listen(ctx, "unix", path)
	if err != nil {
		return "", nil, fmt.Errorf("open the preview: %w", err)
	}

	// The browser's container runs as another user and must be able to connect.
	if err = os.Chmod(path, viewSocketMode); err != nil {
		_ = listener.Close()

		return "", nil, fmt.Errorf("open the preview: %w", err)
	}

	server := &http.Server{
		Handler:           viewHandler{socket: appSocket, base: base, prefix: prefix},
		ReadHeaderTimeout: viewHeaderWait,
	}

	go func() { _ = server.Serve(listener) }()

	return path, func() { _ = server.Close(); _ = os.Remove(path) }, nil
}

type viewHandler struct {
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

	body, _ := io.ReadAll(io.LimitReader(r.Body, maxViewBody))

	out, failure := relay(r.Context(), relayRequest{
		Socket: v.socket, Method: r.Method, Path: rest, Query: r.URL.RawQuery, Body: body, Header: r.Header,
		Base: v.base, Prefix: v.prefix, ClientIP: "127.0.0.1",
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
