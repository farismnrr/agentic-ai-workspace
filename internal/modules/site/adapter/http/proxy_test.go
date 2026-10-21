package http

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

type fakeService struct {
	sites  map[string]domain.Site
	socket string
}

func (f fakeService) List(context.Context, string) ([]domain.Site, error) { return nil, nil }
func (f fakeService) Stop(context.Context, string, string) error          { return nil }
func (f fakeService) Restart(context.Context, string, string) (domain.Site, error) {
	return domain.Site{}, nil
}

func (f fakeService) SetVisibility(context.Context, string, string, string) error {
	return nil
}
func (f fakeService) Socket(string) string { return f.socket }

// The fake issues "9999999999.<holder>" so a test can see that a ticket only works for its holder.
func goodTicket(holder string) string { return "9999999999." + holder }

func (fakeService) Ticket(_, holder string) string { return goodTicket(holder) }
func (fakeService) ValidTicket(token, holder, ticket string) bool {
	return token == "tok" && ticket == goodTicket(holder)
}

func (f fakeService) Resolve(_ context.Context, slug string) (domain.Site, error) {
	if site, ok := f.sites[slug]; ok && site.Status == domain.Running {
		return site, nil
	}

	return domain.Site{}, domain.ErrNotFound
}

func (f fakeService) ResolveOld(_ context.Context, token string) (domain.Site, error) {
	for _, site := range f.sites {
		if site.Token == token && site.Status == domain.Running {
			return site, nil
		}
	}

	return domain.Site{}, domain.ErrNotFound
}

type cookieAuth struct{}

func (cookieAuth) UserID(_ context.Context, token string) (string, error) {
	if token == "owner-session" {
		return "owner", nil
	}

	if token == "other-session" {
		return "other", nil
	}

	return "", domain.ErrNotFound
}

// app is the program a site runs: it records what it was sent.
func startApp(t *testing.T) (socket string, seen chan http.Header) {
	t.Helper()

	dir, err := os.MkdirTemp("", "s")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	socket = filepath.Join(dir, "s.sock")
	seen = make(chan http.Header, 16)

	l, err := (&net.ListenConfig{}).Listen(context.Background(), "unix", socket)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header
		_ = json.NewEncoder(w).Encode(map[string]string{"path": r.URL.Path, "query": r.URL.RawQuery, "host": r.Host})
	})
	mux.HandleFunc("/evil", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Set-Cookie", "session=evil; Path=/")
		w.Header().Set("Content-Security-Policy", "default-src *")
		w.Header().Set("Clear-Site-Data", `"cookies"`)
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<script>alert(1)</script>")
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "sid", Value: "abc", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		})
	})
	mux.HandleFunc("/logout", func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "sid", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		})
	})
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"cookie": r.Header.Get("Cookie")})
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}

	go func() { _ = server.Serve(l) }()

	t.Cleanup(func() { _ = server.Close() })

	return socket, seen
}

func newApp(t *testing.T, visibility domain.Visibility, status domain.Status) (*fiber.App, chan http.Header) {
	t.Helper()

	socket, seen := startApp(t)
	site := domain.Site{
		ProjectID: "p1", UserID: "owner", Slug: "sensio", Token: "tok", Visibility: visibility, Status: status,
	}
	passthrough := func(c fiber.Ctx) error { return c.Next() }

	app := fiber.New()
	NewHandler(fakeService{sites: map[string]domain.Site{"sensio": site}, socket: socket}, cookieAuth{}, false).
		Mount(passthrough, passthrough)(app)
	app.Get("/*", func(c fiber.Ctx) error { return c.SendString("SPA") })

	return app, seen
}

type call struct {
	headers map[string]string
	method  string
	path    string
}

func (c call) do(t *testing.T, app *fiber.App) (int, http.Header, string) {
	t.Helper()

	method := c.method
	if method == "" {
		method = http.MethodGet
	}

	req := httptest.NewRequestWithContext(context.Background(), method, c.path, http.NoBody)
	for name, value := range c.headers {
		req.Header.Set(name, value)
	}

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = res.Body.Close() }()

	body, _ := io.ReadAll(res.Body)

	return res.StatusCode, res.Header, string(body)
}
