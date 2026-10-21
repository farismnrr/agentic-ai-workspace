package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/gofiber/fiber/v3"
)

var errUnknownTestSession = errors.New("unknown session")

func TestDatabaseTestLimitCountsPerSignedInUser(t *testing.T) {
	app := fiber.New()
	app.Post("/test", httpserver.RequireSession(testSessions{"session-a": "user-a", "session-b": "user-b"}),
		databaseTestLimit(), func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

	for range databaseTestsPerMinute {
		if status := databaseRequest(t, app, "/test", "session-a"); status != http.StatusNoContent {
			t.Fatalf("user A request should be allowed before the limit, got %d", status)
		}
	}
	if status := databaseRequest(t, app, "/test", "session-a"); status != http.StatusTooManyRequests {
		t.Fatalf("user A should be rate limited, got %d", status)
	}
	if status := databaseRequest(t, app, "/test", "session-b"); status != http.StatusNoContent {
		t.Fatalf("user B has a separate limit, got %d", status)
	}
}

func TestSharedDatabaseLimitIsSeparateAndCountsPerSignedInUser(t *testing.T) {
	app := fiber.New()
	limiter := httpserver.RequireSession(testSessions{"session-a": "user-a", "session-b": "user-b"})
	app.Post("/test", limiter, databaseTestLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	})
	app.Post("/shared", limiter, databaseSharedLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	})
	for range databaseTestsPerMinute {
		if status := databaseRequest(t, app, "/test", "session-a"); status != http.StatusNoContent {
			t.Fatalf("test endpoint request should be allowed, got %d", status)
		}
	}
	if status := databaseRequest(t, app, "/test", "session-a"); status != http.StatusTooManyRequests {
		t.Fatalf("test endpoint should have its own exhausted limit, got %d", status)
	}
	for range databaseSharedPerMinute {
		if status := databaseRequest(t, app, "/shared", "session-a"); status != http.StatusNoContent {
			t.Fatalf("shared endpoint request should be allowed, got %d", status)
		}
	}
	if status := databaseRequest(t, app, "/shared", "session-a"); status != http.StatusTooManyRequests {
		t.Fatalf("shared endpoint should be rate limited, got %d", status)
	}
	if status := databaseRequest(t, app, "/shared", "session-b"); status != http.StatusNoContent {
		t.Fatalf("shared endpoint should keep separate per-user limits, got %d", status)
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
