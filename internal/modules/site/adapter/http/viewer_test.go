package http

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// appSocket serves a page whose stylesheet link is a root link, as an AI-built app often writes it.
func appSocket(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "v")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	socket := filepath.Join(dir, "s.sock")

	l, err := (&net.ListenConfig{}).Listen(context.Background(), "unix", socket)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/s.css" {
			w.Header().Set("Content-Type", "text/css")
			_, _ = io.WriteString(w, "a{background:url(/i.png)}")

			return
		}

		w.Header().Set("Content-Type", "text/html")
		//nolint:gosec // a test fixture that echoes the header the proxy sets
		_, _ = io.WriteString(w, `<head><link rel="stylesheet" href="/s.css"></head><p>`+r.Header.Get("X-Forwarded-Prefix"))
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}

	go func() { _ = server.Serve(l) }()

	t.Cleanup(func() { _ = server.Close() })

	return socket
}

func viewGet(t *testing.T, socket, path string) (int, string) {
	t.Helper()

	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://view"+path, http.NoBody)

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = res.Body.Close() }()

	body, _ := io.ReadAll(res.Body)

	return res.StatusCode, string(body)
}

func TestViewerShowsTheAppAsThePersonGetsIt(t *testing.T) {
	view, closeView, err := NewViewer().Open(context.Background(), appSocket(t), "/run/shop", "/run/shop/~preview")
	if err != nil {
		t.Fatal(err)
	}

	status, page := viewGet(t, view, "/run/shop/~preview/")
	if status != 200 || !strings.Contains(page, `href="/run/shop/~preview/s.css"`) ||
		!strings.Contains(page, "<p>/run/shop/~preview") {
		t.Fatalf("the page must come with its links under the address: %d %s", status, page)
	}

	if _, css := viewGet(t, view, "/run/shop/~preview/s.css"); css != "a{background:url(/run/shop/~preview/i.png)}" {
		t.Fatalf("stylesheet: %s", css)
	}

	if status, _ = viewGet(t, view, "/s.css"); status != http.StatusNotFound {
		t.Fatalf("a path outside the address must not reach the app, got %d", status)
	}

	closeView()

	if _, err = os.Stat(view); !os.IsNotExist(err) {
		t.Fatal("the preview socket must go away when the picture is done")
	}
}
