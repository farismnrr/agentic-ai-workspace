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

const testDatabasePassword = "password-placeholder"

func TestTestRouteReturnsGenericFailureWithoutCredentials(t *testing.T) {
	store := &testStore{records: map[string]application.StoredConnection{}}
	service := application.NewService(store, testCipher{}, testOwner{}, testEnvironment{}, failedProbe{})
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	NewHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, continueRequest)(app)

	body := `{"type":"postgres","host":"db.example","port":5432,"database":"records",` +
		`"user":"reporter","password":"` + testDatabasePassword + `","tls":true}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/api/projects/p1/db/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name: httpserver.SessionCookieName, Value: "session", Secure: true, HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	result, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
	if strings.Contains(string(result), testDatabasePassword) || strings.Contains(string(result), "postgres://") {
		t.Fatal("test response exposed database credentials")
	}
	if !strings.Contains(string(result), `"category":"auth_failed"`) ||
		!strings.Contains(string(result), "rejected the login") {
		t.Fatalf("failure response is not categorized and generic: %s", result)
	}
}

func TestDeleteMissingConnectionReturnsNotFound(t *testing.T) {
	store := &testStore{records: map[string]application.StoredConnection{}}
	service := application.NewService(store, testCipher{}, testOwner{}, testEnvironment{}, failedProbe{})
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	NewHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, continueRequest)(app)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/projects/p1/db", nil)
	req.AddCookie(&http.Cookie{
		Name: httpserver.SessionCookieName, Value: "session", Secure: true, HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("missing connection delete status = %d, want 404", response.StatusCode)
	}
}

func TestSharedProvisionRouteReturnsMetadataOnly(t *testing.T) {
	store := &testStore{records: map[string]application.StoredConnection{}}
	service := application.NewService(
		store, testCipher{}, testOwner{}, testEnvironment{}, failedProbe{}, routeProvisioner{},
	)
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	NewHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, continueRequest)(app)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects/p1/db/shared", nil)
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
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"managed":true`) ||
		strings.Contains(string(body), testDatabasePassword) || strings.Contains(string(body), "postgres://") {
		t.Fatalf("unexpected shared provision response: status=%d body=%s", response.StatusCode, body)
	}
}

type routeProvisioner struct{}

func (routeProvisioner) Configured(context.Context) (bool, error) { return true, nil }
func (routeProvisioner) Provision(context.Context, string) (application.ProvisionResult, error) {
	return application.ProvisionResult{
		Host: "db.local", Port: 5432, Database: "records", Role: "project", Password: testDatabasePassword,
	}, nil
}
func (routeProvisioner) Restore(context.Context, string, string) error { return nil }

type testStore struct {
	records map[string]application.StoredConnection
}

func (s *testStore) Get(_ context.Context, id string) (application.StoredConnection, error) {
	item, ok := s.records[id]
	if !ok {
		return item, domain.ErrMissing
	}
	return item, nil
}

func (s *testStore) Save(_ context.Context, id string, item application.StoredConnection) error {
	s.records[id] = item
	return nil
}
func (s *testStore) Delete(_ context.Context, id string) error { delete(s.records, id); return nil }

type testCipher struct{}

func (testCipher) Encrypt(value []byte) (string, error) { return "sealed:" + string(value), nil }
func (testCipher) Decrypt(value string) ([]byte, error) {
	return []byte(strings.TrimPrefix(value, "sealed:")), nil
}

type testOwner struct{}

func (testOwner) Owner(_ context.Context, id string) (string, string, error) {
	if id == "p1" {
		return "u1", "", nil
	}
	return "", "", domain.ErrMissing
}

type testEnvironment struct{}

func (testEnvironment) SetSecret(context.Context, string, string, string) error { return nil }
func (testEnvironment) DeleteSecret(context.Context, string, string) error      { return nil }
func (testEnvironment) Values(context.Context, string) (map[string]string, error) {
	return map[string]string{}, nil
}

type failedProbe struct{}

func (failedProbe) Probe(context.Context, domain.Connection) error {
	return domain.ProbeError{Category: domain.AuthFailed}
}

type testSessions struct{}

func (testSessions) UserID(_ context.Context, token string) (string, error) {
	if token == "session" {
		return "u1", nil
	}
	return "", domain.ErrMissing
}
