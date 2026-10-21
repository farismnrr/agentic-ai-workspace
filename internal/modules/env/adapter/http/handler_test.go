package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

func TestHandlerScopesOwnersMasksSecretsAndReportsInvalidName(t *testing.T) {
	store := &handlerStore{items: map[string]application.StoredVariable{
		"p1:SECRET_KEY": {Name: "SECRET_KEY", Sealed: "sealed:private-value", IsSecret: true},
		"p1:PUBLIC_URL": {Name: "PUBLIC_URL", Sealed: "sealed:https://example.test", IsSecret: false},
	}}
	service := application.NewService(store, handlerCipher{}, handlerOwners{"p1": "u1", "p2": "u2"}, time.Now)
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	NewHandler(service).Mount(httpserver.RequireSession(handlerSessions{}), continueRequest)(app)

	status, _ := callHandler(t, app, http.MethodGet, "/api/projects/p2/env", "", "session")
	if status != http.StatusNotFound {
		t.Fatalf("other owner's project status = %d, want 404", status)
	}

	status, body := callHandler(t, app, http.MethodGet, "/api/projects/p1/env", "", "session")
	if status != http.StatusOK {
		t.Fatalf("list status = %d, want 200: %s", status, body)
	}
	if strings.Contains(body, "private-value") || !strings.Contains(body, `"name":"SECRET_KEY"`) {
		t.Fatalf("secret list response is not masked: %s", body)
	}
	if !strings.Contains(body, `"value":"https://example.test"`) {
		t.Fatalf("non-secret value missing from list: %s", body)
	}

	status, body = callHandler(t, app, http.MethodPut, "/api/projects/p1/env",
		`{"name":"bad-name","value":"x"}`, "session")
	if status != http.StatusBadRequest || !strings.Contains(body, `"error":"invalid_name"`) {
		t.Fatalf("invalid name response = %d %s, want 400 invalid_name", status, body)
	}
}

func callHandler(t *testing.T, app *fiber.App, method, path, body, session string) (int, string) {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if method == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{
		Name: httpserver.SessionCookieName, Value: session, Secure: true, HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(data)
}

type handlerStore struct {
	items map[string]application.StoredVariable
}

func (s *handlerStore) List(_ context.Context, projectID string) ([]application.StoredVariable, error) {
	items := make([]application.StoredVariable, 0)
	for key, item := range s.items {
		if strings.HasPrefix(key, projectID+":") {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *handlerStore) Put(_ context.Context, projectID string, item application.StoredVariable) error {
	s.items[projectID+":"+item.Name] = item
	return nil
}

func (s *handlerStore) Delete(_ context.Context, projectID, name string) error {
	delete(s.items, projectID+":"+name)
	return nil
}

type handlerCipher struct{}

func (handlerCipher) Encrypt(value []byte) (string, error) { return "sealed:" + string(value), nil }
func (handlerCipher) Decrypt(value string) ([]byte, error) {
	return []byte(strings.TrimPrefix(value, "sealed:")), nil
}

type handlerOwners map[string]string

func (o handlerOwners) Owner(_ context.Context, projectID string) (string, string, error) {
	owner, ok := o[projectID]
	if !ok {
		return "", "", domain.ErrNotFound
	}
	return owner, "project", nil
}

type handlerSessions struct{}

func (handlerSessions) UserID(_ context.Context, token string) (string, error) {
	if token == "session" {
		return "u1", nil
	}
	return "", domain.ErrNotFound
}
