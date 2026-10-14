package http

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func TestAppSeesNeitherTheLoginNorThePrefix(t *testing.T) {
	app, seen := newApp(t, domain.Public, domain.Running)
	headers := as("document", "owner-session")
	headers["Authorization"] = "Bearer abc"
	headers["X-Forwarded-For"] = "6.6.6.6"

	status, _, body := (call{path: "/run/sensio/tok/echo?x=1", headers: headers}).do(t, app)
	if status != 200 || !strings.Contains(body, `"path":"/echo"`) || !strings.Contains(body, `"query":"x=1"`) {
		t.Fatalf("status %d body %s", status, body)
	}

	got := <-seen
	for _, name := range []string{"Cookie", "Authorization"} {
		if got.Get(name) != "" {
			t.Errorf("the app must not receive %s, got %q", name, got.Get(name))
		}
	}

	if got.Get("X-Forwarded-Prefix") != "/run/sensio/tok" || got.Get("X-Forwarded-For") == "6.6.6.6" {
		t.Errorf("proxy headers wrong: %v", got)
	}
}

func TestEverythingFromTheAppIsSandboxedAndCannotSetCookies(t *testing.T) {
	app, _ := newApp(t, domain.Public, domain.Running)
	status, header, _ := (call{path: "/run/sensio/tok/evil", headers: as("document", "")}).do(t, app)

	if status != 200 {
		t.Fatalf("status %d", status)
	}

	csp := header.Get("Content-Security-Policy")
	if !strings.HasPrefix(csp, "sandbox ") || strings.Contains(csp, "allow-same-origin") ||
		strings.Contains(csp, "default-src") {
		t.Errorf("csp %q", csp)
	}

	for _, name := range []string{"Set-Cookie", "Clear-Site-Data"} {
		if header.Get(name) != "" {
			t.Errorf("%s must not reach the browser", name)
		}
	}

	if header.Get("X-Content-Type-Options") != "nosniff" || header.Get("Cache-Control") != "no-store" {
		t.Errorf("headers %v", header)
	}
}

func TestRedirectsStayUnderTheSitePathAndNamesAreCanonical(t *testing.T) {
	app, _ := newApp(t, domain.Public, domain.Running)

	status, header, _ := (call{path: "/run/sensio/tok/redirect", headers: as("document", "")}).do(t, app)
	if status != http.StatusFound || header.Get("Location") != "/run/sensio/tok/login" {
		t.Errorf("app redirect: %d %q", status, header.Get("Location"))
	}

	for path, want := range map[string]string{
		"/run/other-name/tok/echo?a=b": "/run/sensio/tok/echo?a=b",
		"/run/sensio/tok":              "/run/sensio/tok/",
	} {
		status, header, _ = (call{path: path, headers: as("document", "")}).do(t, app)
		if status != http.StatusFound || header.Get("Location") != want {
			t.Errorf("%s: %d %q, want %q", path, status, header.Get("Location"), want)
		}
	}
}

func TestServiceWorkersAndPreflightsAreHandledByTheProxy(t *testing.T) {
	app, seen := newApp(t, domain.Public, domain.Running)

	worker := map[string]string{"Service-Worker": "script"}
	if got, _, _ := (call{path: "/run/sensio/tok/echo", headers: worker}).do(t, app); got != 403 {
		t.Errorf("service worker script fetch: %d", got)
	}

	ask := map[string]string{"Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "content-type"}
	status, header, _ := (call{method: http.MethodOptions, path: "/run/sensio/tok/echo", headers: ask}).do(t, app)

	if status != 204 || header.Get("Access-Control-Allow-Origin") != "*" ||
		header.Get("Access-Control-Allow-Headers") != "content-type" {
		t.Errorf("preflight: %d %v", status, header)
	}

	if len(seen) != 0 {
		t.Error("a preflight must not reach the app")
	}
}

func TestAnAppThatIsDownIsABadGateway(t *testing.T) {
	running := domain.Site{Slug: "sensio", Token: "tok", Visibility: domain.Public, Status: domain.Running}
	service := fakeService{
		sites:  map[string]domain.Site{"tok": running},
		socket: filepath.Join(t.TempDir(), "missing.sock"),
	}
	down := fiber.New()
	pass := func(c fiber.Ctx) error { return c.Next() }
	NewHandler(service, cookieAuth{}, false).Mount(pass, pass)(down)

	if got, _, _ := (call{path: "/run/sensio/tok/echo", headers: as("document", "")}).do(t, down); got != 502 {
		t.Fatalf("status %d", got)
	}
}
