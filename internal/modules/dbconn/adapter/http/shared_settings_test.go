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
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

func TestSharedSettingsGetAndPutNeverReturnPassword(t *testing.T) {
	store := &sharedSettingsHTTPStore{}
	body := `{"host":"db.example","port":5432,"database":"workspace","user":"admin",` +
		`"password":"never-return-this","sslMode":"verify-full"}`
	app := adminSettingsApp(store)
	putBody := requestAdminSettings(t, app, http.MethodPut, body)
	if strings.Contains(string(putBody), "never-return-this") {
		t.Fatal("PUT response exposed password")
	}
	getBody := requestAdminSettings(t, app, http.MethodGet, "")
	if strings.Contains(string(getBody), "never-return-this") || !strings.Contains(string(getBody), `"passwordSet":true`) {
		t.Fatalf("GET response did not safely report saved password: %s", getBody)
	}
	store.record.CACertSealed = "sealed:PEM-MUST-NOT-LEAK"
	secretBody := requestAdminSettings(t, app, http.MethodGet, "")
	if strings.Contains(string(secretBody), "PEM-MUST-NOT-LEAK") ||
		!strings.Contains(string(secretBody), `"caSet":true`) {
		t.Fatalf("GET exposed PEM or omitted set state: %s", secretBody)
	}
}

func adminSettingsApp(store *sharedSettingsHTTPStore) *fiber.App {
	service := application.NewSharedSettingsService(store, testCipher{})
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	admin := func(c fiber.Ctx) error { httpserver.SetRole(c, "admin"); return c.Next() }
	NewSharedSettingsHandler(service).Mount(
		httpserver.RequireSession(testSessions{}), continueRequest, admin, admin,
	)(app)
	return app
}

func requestAdminSettings(t *testing.T, app *fiber.App, method, body string) []byte {
	t.Helper()
	var requestBody io.Reader
	if body != "" {
		requestBody = strings.NewReader(body)
	}
	request := httptest.NewRequestWithContext(
		context.Background(), method, "/api/settings/database", requestBody,
	)
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(secureSessionCookie())
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	responseBody, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if readErr != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("%s status=%d err=%v body=%s", method, response.StatusCode, readErr, responseBody)
	}
	return responseBody
}

func TestSharedSettingsRejectInvalidPortAndSSLMode(t *testing.T) {
	service := application.NewSharedSettingsService(&sharedSettingsHTTPStore{}, testCipher{})
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	admin := func(c fiber.Ctx) error { httpserver.SetRole(c, "admin"); return c.Next() }
	NewSharedSettingsHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, admin, admin)(app)
	for _, body := range []string{
		`{"host":"db","port":70000,"database":"workspace","user":"admin","password":"secret","sslMode":"prefer"}`,
		`{"host":"db","port":5432,"database":"workspace","user":"admin","password":"secret","sslMode":"unknown"}`,
		`{"host":"db","port":5432,"database":"workspace","user":"admin",` +
			`"password":"secret","sslMode":"prefer","caCert":"bad PEM"}`,
	} {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPut,
			"/api/settings/database", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(secureSessionCookie())
		response, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("invalid settings status = %d, want 400", response.StatusCode)
		}
	}
}

func TestSharedSettingsNonAdminGetsConfiguredOnlyAndCannotSave(t *testing.T) {
	store := &sharedSettingsHTTPStore{record: application.StoredSharedSettings{
		PasswordSealed: "sealed:password", Settings: domain.SharedSettings{Kind: "postgres", Host: "db.example"},
	}}
	service := application.NewSharedSettingsService(store, testCipher{})
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	user := func(c fiber.Ctx) error { httpserver.SetRole(c, "user"); return c.Next() }
	adminOnly := func(c fiber.Ctx) error { return c.SendStatus(http.StatusForbidden) }
	NewSharedSettingsHandler(service).Mount(
		httpserver.RequireSession(testSessions{}), continueRequest, user, adminOnly,
	)(app)
	for _, path := range []string{"/api/settings/database", "/api/settings/database/postgres"} {
		get := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
		get.AddCookie(secureSessionCookie())
		response, err := app.Test(get)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK || string(body) != `{"configured":true}` {
			t.Fatalf("non-admin GET %s = %d %s", path, response.StatusCode, body)
		}
	}
	put := httptest.NewRequestWithContext(
		context.Background(), http.MethodPut, "/api/settings/database", strings.NewReader(`{}`),
	)
	put.Header.Set("Content-Type", "application/json")
	put.AddCookie(secureSessionCookie())
	response, err := app.Test(put)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("non-admin PUT status = %d", response.StatusCode)
	}
}

func secureSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name: httpserver.SessionCookieName, Value: "session", Secure: true,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	}
}

type sharedSettingsHTTPStore struct {
	record application.StoredSharedSettings
}

func (s *sharedSettingsHTTPStore) Get(_ context.Context, _ string) (application.StoredSharedSettings, error) {
	if s.record.PasswordSealed == "" {
		return application.StoredSharedSettings{}, application.ErrSharedSettingsMissing
	}
	return s.record, nil
}

func (s *sharedSettingsHTTPStore) Save(_ context.Context, _ string, record application.StoredSharedSettings) error {
	s.record = record
	s.record.Settings.Password = ""
	return nil
}
