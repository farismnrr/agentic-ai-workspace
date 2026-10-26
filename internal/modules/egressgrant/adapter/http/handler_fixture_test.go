package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	accountdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/gofiber/fiber/v3"
)

var (
	errTestSession  = errors.New("no session")
	errTestNotFound = errors.New("not found")
	errTestTask     = errors.New("invalid task")
)

type testAuth struct{}

func (testAuth) UserID(_ context.Context, token string) (string, error) {
	if token == "owner" || token == "other" || token == "admin" {
		return token, nil
	}
	return "", errTestSession
}

type testOwner struct{}

func (testOwner) Owner(_ context.Context, project string) (string, string, error) {
	if project == "p1" {
		return "owner", "p1", nil
	}
	return "", "", errTestNotFound
}

type testRoles struct{}

func (testRoles) Profile(_ context.Context, id string) (accountdomain.User, error) {
	role := accountdomain.RoleUser
	if id == "admin" {
		role = accountdomain.RoleAdmin
	}
	return accountdomain.User{ID: id, Role: role}, nil
}

type testService struct{ grant domain.Grant }

func (s *testService) Ask(_ context.Context, project, task, host, reason string,
	by domain.Actor,
) (domain.Grant, bool, error) {
	if task != "t1" {
		return domain.Grant{}, false, errTestTask
	}
	g, e := domain.Request("g1", project, task, host, reason, by, time.Now())
	if e == nil {
		s.grant = g
	}
	return g, false, wrapTest(e)
}

func (s *testService) Approve(_ context.Context, _ string, by domain.Decider, ttl time.Duration) (domain.Grant, error) {
	g, e := s.grant.Approve(by, ttl, time.Now())
	if e == nil {
		s.grant = g
	}
	return g, wrapTest(e)
}

func (s *testService) Deny(_ context.Context, _ string, by domain.Decider) (domain.Grant, error) {
	g, e := s.grant.Deny(by, time.Now())
	if e == nil {
		s.grant = g
	}
	return g, wrapTest(e)
}

func (s *testService) Revoke(_ context.Context, _ string, by domain.Decider) (domain.Grant, error) {
	g, e := s.grant.Revoke(by, time.Now())
	if e == nil {
		s.grant = g
	}
	return g, wrapTest(e)
}

func (s *testService) Get(_ context.Context, id string) (domain.Grant, error) {
	if id != s.grant.ID {
		return domain.Grant{}, errTestNotFound
	}
	return s.grant, nil
}

func (s *testService) ListProject(context.Context, string) ([]domain.Grant, error) {
	return []domain.Grant{s.grant}, nil
}

func wrapTest(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("test grant: %w", err)
}

func testApp(t *testing.T) (*fiber.App, *testService) {
	t.Helper()
	csrf, err := NewCSRF()
	if err != nil {
		t.Fatal(err)
	}
	s := &testService{grant: domain.Grant{
		ID: "g1", ProjectID: "p1", TaskID: "t1", Host: "example.com",
		Reason: "need it", Status: domain.StatusPending, ExpiresAt: time.Now().Add(time.Hour),
	}}
	app := fiber.New()
	NewHandler(s, testOwner{}, testRoles{}, csrf).Mount(
		httpserver.RequireSession(testAuth{}), httpserver.SameOrigin("https://app.example", nil))(app)
	return app, s
}

type reply struct {
	body []byte
	code int
}

func send(t *testing.T, app *fiber.App, method, path, user, origin, token, body string) reply {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	req.AddCookie(&http.Cookie{
		Name: "session", Value: user, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if token != "" {
		req.Header.Set("X-Egress-CSRF", token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return reply{code: resp.StatusCode, body: data}
}

func issue(t *testing.T, app *fiber.App, user string) string {
	t.Helper()
	resp := send(t, app, "GET", "/api/projects/p1/egress-grants/csrf", user, "", "", "")
	if resp.code != http.StatusOK {
		t.Fatalf("csrf status %d", resp.code)
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(bytes.NewReader(resp.body)).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Token
}
