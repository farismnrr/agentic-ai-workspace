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
		if status := databaseTestRequest(t, app, "session-a"); status != http.StatusNoContent {
			t.Fatalf("user A request should be allowed before the limit, got %d", status)
		}
	}
	if status := databaseTestRequest(t, app, "session-a"); status != http.StatusTooManyRequests {
		t.Fatalf("user A should be rate limited, got %d", status)
	}
	if status := databaseTestRequest(t, app, "session-b"); status != http.StatusNoContent {
		t.Fatalf("user B has a separate limit, got %d", status)
	}
}

func databaseTestRequest(t *testing.T, app *fiber.App, session string) int {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", http.NoBody)
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
