package dbconn_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	dbconnhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/http"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

var errUnknownTestSession = errors.New("unknown session")

func TestDatabaseTestLimitCountsPerSignedInUser(t *testing.T) {
	app := fiber.New()
	app.Post("/test", httpserver.RequireSession(testSessions{"session-a": "user-a", "session-b": "user-b"}),
		dbconnhttp.TestRateLimit(), func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

	for range 5 {
		requireRateStatus(t, databaseRequest(t, app, "/test", "session-a"),
			http.StatusNoContent, "user A request should be allowed before the limit")
	}
	requireRateStatus(t, databaseRequest(t, app, "/test", "session-a"),
		http.StatusTooManyRequests, "user A should be rate limited")
	requireRateStatus(t, databaseRequest(t, app, "/test", "session-b"),
		http.StatusNoContent, "user B has a separate limit")
}

func TestSharedDatabaseLimitIsSeparateAndCountsPerSignedInUser(t *testing.T) {
	app := fiber.New()
	limiter := httpserver.RequireSession(testSessions{"session-a": "user-a", "session-b": "user-b"})
	app.Post("/test", limiter, dbconnhttp.TestRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	})
	app.Post("/shared", limiter, dbconnhttp.SharedRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	})
	for range 5 {
		requireRateStatus(t, databaseRequest(t, app, "/test", "session-a"),
			http.StatusNoContent, "test endpoint request should be allowed")
	}
	requireRateStatus(t, databaseRequest(t, app, "/test", "session-a"),
		http.StatusTooManyRequests, "test endpoint should have its own exhausted limit")
	for range 10 {
		requireRateStatus(t, databaseRequest(t, app, "/shared", "session-a"),
			http.StatusNoContent, "shared endpoint request should be allowed")
	}
	requireRateStatus(t, databaseRequest(t, app, "/shared", "session-a"),
		http.StatusTooManyRequests, "shared endpoint should be rate limited")
	requireRateStatus(t, databaseRequest(t, app, "/shared", "session-b"),
		http.StatusNoContent, "shared endpoint should keep separate per-user limits")
}

func requireRateStatus(t *testing.T, got, want int, message string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s, got %d want %d", message, got, want)
	}
}

func databaseRequest(t *testing.T, app *fiber.App, path, session string) int {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, http.NoBody)
	req.AddCookie(&http.Cookie{
		Name: httpserver.SessionCookieName, Value: session, Secure: true, HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	return res.StatusCode
}

type testSessions map[string]string

func (s testSessions) UserID(_ context.Context, token string) (string, error) {
	user, ok := s[token]
	if !ok {
		return "", errUnknownTestSession
	}
	return user, nil
}
