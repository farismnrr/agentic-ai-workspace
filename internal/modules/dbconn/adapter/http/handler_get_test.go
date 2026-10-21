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

func TestGetIncludesManagedAndSharedPostgresFlags(t *testing.T) {
	store := &testStore{records: map[string]application.StoredConnection{}}
	service := application.NewService(
		store, testCipher{}, testOwner{}, testEnvironment{}, failedProbe{}, routeProvisioner{},
	)
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	NewHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, continueRequest)(app)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects/p1/db", nil)
	req.AddCookie(&http.Cookie{
		Name: httpserver.SessionCookieName, Value: "session", Secure: true, HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
	if got := string(body); !strings.Contains(got, `"managed":false`) ||
		!strings.Contains(got, `"sharedPostgres":true`) {
		t.Fatalf("GET response missing capability flags: %s", body)
	}
}

func TestGetReturnsSQLiteChoiceWhenNoConnectionExists(t *testing.T) {
	service := application.NewService(&testStore{records: map[string]application.StoredConnection{}},
		testCipher{}, testOwner{}, testEnvironment{}, failedProbe{})
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	NewHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, continueRequest)(app)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects/p1/db", nil)
	req.AddCookie(secureSessionCookie())
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"choice":"sqlite"`) {
		t.Fatalf("SQLite GET response = %d %s", response.StatusCode, body)
	}
}
