package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

func TestKindsSummaryIsAvailableToSignedInNonAdmin(t *testing.T) {
	service := application.NewSharedSettingsService(&sharedSettingsHTTPStore{
		record: application.StoredSharedSettings{PasswordSealed: "sealed:password"},
	}, testCipher{})
	app := fiber.New()
	user := func(c fiber.Ctx) error { httpserver.SetRole(c, "user"); return c.Next() }
	adminOnly := func(c fiber.Ctx) error { return c.SendStatus(http.StatusForbidden) }
	NewSharedSettingsHandler(service).Mount(
		httpserver.RequireSession(testSessions{}), func(c fiber.Ctx) error { return c.Next() }, user, adminOnly,
	)(app)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/databases/kinds", nil)
	req.AddCookie(secureSessionCookie())
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	want := `[{"admin":false,"configured":true,"kind":"postgres"},` +
		`{"admin":false,"configured":true,"kind":"mysql"},` +
		`{"admin":false,"configured":true,"kind":"mongodb"}]`
	if response.StatusCode != http.StatusOK || string(body) != want {
		t.Fatalf("kind summary = %d %s", response.StatusCode, body)
	}
}

func TestKindsSummaryMarksAdministrators(t *testing.T) {
	service := application.NewSharedSettingsService(&sharedSettingsHTTPStore{}, testCipher{})
	app := fiber.New()
	admin := func(c fiber.Ctx) error { httpserver.SetRole(c, "admin"); return c.Next() }
	NewSharedSettingsHandler(service).Mount(
		httpserver.RequireSession(testSessions{}), func(c fiber.Ctx) error { return c.Next() }, admin, admin,
	)(app)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/databases/kinds", nil)
	req.AddCookie(secureSessionCookie())
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"admin":true`) {
		t.Fatalf("admin kind summary = %d %s", response.StatusCode, body)
	}
}

func TestUnknownSettingsKindReturnsNotFound(t *testing.T) {
	app := adminSettingsApp(&sharedSettingsHTTPStore{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/settings/database/oracle", nil)
	req.AddCookie(secureSessionCookie())
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown kind status = %d, want 404", response.StatusCode)
	}
}

func TestKindSettingsRouteReturnsTLSChoiceAndSetFlags(t *testing.T) {
	app := adminSettingsApp(&sharedSettingsHTTPStore{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut,
		"/api/settings/database/mysql", strings.NewReader(
			`{"host":"db.example","database":"app","user":"admin","password":"secret","tls":"ca"}`,
		))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(secureSessionCookie())
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"kind":"mysql"`) ||
		!strings.Contains(string(body), `"tls":"ca"`) || !strings.Contains(string(body), `"passwordSet":true`) ||
		strings.Contains(string(body), "sslMode") {
		t.Fatalf("kind settings response = %d %s", response.StatusCode, body)
	}
}

func TestUnconfiguredKindIncludesItsIdentityAndDefaultPort(t *testing.T) {
	app := adminSettingsApp(&sharedSettingsHTTPStore{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
		"/api/settings/database/mongodb", nil)
	req.AddCookie(secureSessionCookie())
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	for _, field := range []string{`"kind":"mongodb"`, `"port":27017`, `"tls":"none"`, `"configured":false`} {
		if !strings.Contains(string(body), field) {
			t.Fatalf("unconfigured response omitted %s: %s", field, body)
		}
	}
}
