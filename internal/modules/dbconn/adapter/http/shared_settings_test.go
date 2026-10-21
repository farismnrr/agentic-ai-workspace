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

func TestSharedSettingsGetAndPutNeverReturnPassword(t *testing.T) {
	store := &sharedSettingsHTTPStore{}
	service := application.NewSharedSettingsService(store, testCipher{})
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	admin := func(c fiber.Ctx) error { httpserver.SetRole(c, "admin"); return c.Next() }
	NewSharedSettingsHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, admin, admin)(app)

	body := `{"host":"db.example","port":5432,"database":"workspace","user":"admin",` +
		`"password":"never-return-this","sslMode":"verify-full"}`
	put := httptest.NewRequestWithContext(context.Background(), http.MethodPut,
		"/api/settings/database", strings.NewReader(body))
	put.Header.Set("Content-Type", "application/json")
	put.AddCookie(secureSessionCookie())
	putResponse, err := app.Test(put)
	if err != nil {
		t.Fatal(err)
	}
	putBody, err := io.ReadAll(putResponse.Body)
	_ = putResponse.Body.Close()
	if err != nil || putResponse.StatusCode != http.StatusOK {
		t.Fatalf("PUT status=%d err=%v body=%s", putResponse.StatusCode, err, putBody)
	}
	if strings.Contains(string(putBody), "never-return-this") {
		t.Fatal("PUT response exposed password")
	}

	get := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/settings/database", nil)
	get.AddCookie(secureSessionCookie())
	getResponse, err := app.Test(get)
	if err != nil {
		t.Fatal(err)
	}
	getBody, err := io.ReadAll(getResponse.Body)
	_ = getResponse.Body.Close()
	if err != nil || getResponse.StatusCode != http.StatusOK {
		t.Fatalf("GET status=%d err=%v body=%s", getResponse.StatusCode, err, getBody)
	}
	if strings.Contains(string(getBody), "never-return-this") || !strings.Contains(string(getBody), `"passwordSet":true`) {
		t.Fatalf("GET response did not safely report saved password: %s", getBody)
	}
}

func TestSharedSettingsRejectInvalidPortAndSSLMode(t *testing.T) {
	service := application.NewSharedSettingsService(&sharedSettingsHTTPStore{}, testCipher{})
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	admin := func(c fiber.Ctx) error { httpserver.SetRole(c, "admin"); return c.Next() }
	NewSharedSettingsHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, admin, admin)(app)
	for _, body := range []string{
		`{"host":"db","port":0,"database":"workspace","user":"admin","password":"secret","sslMode":"prefer"}`,
		`{"host":"db","port":5432,"database":"workspace","user":"admin","password":"secret","sslMode":"unknown"}`,
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

func secureSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name: httpserver.SessionCookieName, Value: "session", Secure: true,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	}
}

type sharedSettingsHTTPStore struct {
	record application.StoredSharedSettings
}

func (s *sharedSettingsHTTPStore) Get(context.Context) (application.StoredSharedSettings, error) {
	if s.record.PasswordSealed == "" {
		return application.StoredSharedSettings{}, application.ErrSharedSettingsMissing
	}
	return s.record, nil
}

func (s *sharedSettingsHTTPStore) Save(_ context.Context, record application.StoredSharedSettings) error {
	s.record = record
	s.record.Settings.Password = ""
	return nil
}
