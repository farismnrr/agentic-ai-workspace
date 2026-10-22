package http

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/publicsuffix"
)

func protectedAppSocket(t *testing.T) string {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "s.sock")
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			if r.Form.Get("csrf") != "token" || r.Header.Get("Cookie") != "csrf=token" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			http.SetCookie(w, testCookie("auth", "html"))
			return
		}
		http.SetCookie(w, testCookie("csrf", "token"))
		_, _ = io.WriteString(w, `<form action="/submit" method="post"><input name="csrf" value="token"></form>`)
	})
	mux.HandleFunc("/private/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Cookie"), "auth=") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, "protected")
	})
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Origin") != "null" ||
			r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		http.SetCookie(w, testCookie("auth", "json"))
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return socket
}

func testCookie(name, value string) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
	}
}

func viewerRequest(t *testing.T, socket, method, path, contentType, body string) (int, http.Header) {
	t.Helper()
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(t.Context(), method, "http://view"+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if method == http.MethodPost || method == http.MethodOptions {
		req.Header.Set("Origin", "null")
	}
	if method == http.MethodOptions {
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "content-type")
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, res.Body)
	return res.StatusCode, res.Header
}

func fixtureJar(t *testing.T) *cookiejar.Jar {
	t.Helper()
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		t.Fatal(err)
	}
	return jar
}

func TestViewerRelaysCookieAndJSONLoginWithoutChangingOrigin(t *testing.T) {
	appSocket := protectedAppSocket(t)
	formJar, jsonJar := fixtureJar(t), fixtureJar(t)
	formView, closeForm, err := NewViewer().Open(context.Background(), appSocket, "/run/shop", "/run/shop", formJar)
	if err != nil {
		t.Fatal(err)
	}
	defer closeForm()
	jsonView, closeJSON, err := NewViewer().Open(context.Background(), appSocket, "/run/shop", "/run/shop", jsonJar)
	if err != nil {
		t.Fatal(err)
	}
	defer closeJSON()

	assertFormSession(t, formView)
	assertJsonSession(t, jsonView)
}

func assertFormSession(t *testing.T, view string) {
	t.Helper()
	status, _ := viewerRequest(t, view, http.MethodGet, "/run/shop/login", "", "")
	if status != http.StatusOK {
		t.Fatalf("login form status = %d", status)
	}
	status, headers := viewerRequest(t, view, http.MethodPost, "/run/shop/login",
		"application/x-www-form-urlencoded", "csrf=token")
	if status != http.StatusOK || headers.Get("Set-Cookie") != "" {
		t.Fatalf("cookie plus CSRF login status = %d", status)
	}
	assertProtectedRoutes(t, view, "form", http.StatusOK)
}

func assertJsonSession(t *testing.T, view string) {
	t.Helper()
	status, _ := viewerRequest(t, view, http.MethodGet, "/run/shop/private/one", "", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("separate session jar status = %d", status)
	}
	status, preflight := viewerRequest(t, view, http.MethodOptions, "/run/shop/api/login", "", "")
	if status != http.StatusNoContent || preflight.Get("Access-Control-Allow-Origin") != "*" ||
		!strings.Contains(preflight.Get("Access-Control-Allow-Methods"), "POST") {
		t.Fatalf("preflight response = %d %v", status, preflight)
	}
	status, headers := viewerRequest(t, view, http.MethodPost, "/run/shop/api/login",
		"application/json", `{"email":"a","password":"b"}`)
	if status != http.StatusOK || headers.Get("Set-Cookie") != "" {
		t.Fatalf("JSON login status = %d", status)
	}
	assertProtectedRoutes(t, view, "JSON", http.StatusOK)
}

func assertProtectedRoutes(t *testing.T, view, label string, expected int) {
	t.Helper()
	for _, path := range []string{"/run/shop/private/one", "/run/shop/private/two"} {
		status, _ := viewerRequest(t, view, http.MethodGet, path, "", "")
		if status != expected {
			t.Fatalf("protected %s route %s status = %d", label, path, status)
		}
	}
}
