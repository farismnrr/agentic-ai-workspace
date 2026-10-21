package http

import (
	"context"
	"errors"
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

func TestSharedProvisionRouteRedactsProvisionerErrors(t *testing.T) {
	store := &testStore{records: map[string]application.StoredConnection{}}
	service := application.NewService(store, testCipher{}, testOwner{}, testEnvironment{}, failedProbe{},
		errorProvisioner{err: errSecretProvisionFailure})
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
	if response.StatusCode != http.StatusInternalServerError || !strings.Contains(string(body), "provision_failed") ||
		strings.Contains(string(body), "admin-secret") || strings.Contains(string(body), "project-secret") {
		t.Fatalf("unsafe provision error response: status=%d body=%s", response.StatusCode, body)
	}
}

func TestSharedProvisionRouteMapsManualConnectionConflict(t *testing.T) {
	store := &testStore{records: map[string]application.StoredConnection{}}
	manual := application.StoredConnection{Connection: domain.Connection{
		Kind: domain.Postgres, Host: "db.local", Port: 5432, Database: "records",
		User: "manual", EnvName: "DATABASE_URL",
	}}
	store.records["p1"] = manual
	service := application.NewService(store, testCipher{}, testOwner{}, testEnvironment{}, failedProbe{},
		errorProvisioner{err: errSecretProvisionFailure})
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
	if response.StatusCode != http.StatusConflict || !strings.Contains(string(body), "manual_connection_exists") {
		t.Fatalf("manual connection conflict = status %d, body %s", response.StatusCode, body)
	}
}

func TestSharedProvisionFailureStatusCodes(t *testing.T) {
	for _, test := range []struct {
		category application.ProvisionFailure
		status   int
	}{
		{application.AdminRejected, http.StatusBadGateway},
		{application.InsufficientPrivileges, http.StatusBadGateway},
		{application.AdminNotConfigured, http.StatusServiceUnavailable},
		{application.AdminURLInvalid, http.StatusServiceUnavailable},
		{application.LocalSaveFailed, http.StatusInternalServerError},
		{application.EnvironmentWriteFailed, http.StatusInternalServerError},
		{application.CompensationFailed, http.StatusInternalServerError},
		{application.EnvironmentNameInUse, http.StatusConflict},
	} {
		t.Run(string(test.category), func(t *testing.T) {
			if got := provisionStatus(application.ProvisionError{Category: test.category}); got != test.status {
				t.Fatalf("status for %s = %d, want %d", test.category, got, test.status)
			}
		})
	}
}

func TestSharedProvisionRemoteFailureReturnsBadGateway(t *testing.T) {
	failure := application.ProvisionError{Category: application.ProvisionFailed, Remote: true}
	if got := provisionStatus(failure); got != http.StatusBadGateway {
		t.Fatalf("remote provisioning status = %d, want %d", got, http.StatusBadGateway)
	}
	failure = application.ProvisionError{Category: application.ProvisionFailed}
	if got := provisionStatus(failure); got != http.StatusInternalServerError {
		t.Fatalf("local provisioning status = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestSharedProvisionRemoteCompensationFailureReturnsBadGateway(t *testing.T) {
	failure := application.ProvisionError{Category: application.CompensationFailed, Remote: true}
	if got := provisionStatus(failure); got != http.StatusBadGateway {
		t.Fatalf("remote compensation status = %d, want %d", got, http.StatusBadGateway)
	}
}

func TestProvisionInProgressReturnsConflictForDelete(t *testing.T) {
	for _, method := range []string{http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			store := &busyStore{}
			service := application.NewService(store, testCipher{}, testOwner{}, testEnvironment{}, failedProbe{})
			app := fiber.New()
			continueRequest := func(c fiber.Ctx) error { return c.Next() }
			NewHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, continueRequest)(app)
			body := `{"type":"postgres","host":"db.local","port":5432,"database":"records",` +
				`"user":"reporter","password":"safe-password","tls":true}`
			req := httptest.NewRequestWithContext(context.Background(), method,
				"/api/projects/p1/db", strings.NewReader(body))
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
			if response.StatusCode != http.StatusConflict ||
				!strings.Contains(string(result), `"error":"provision_in_progress"`) {
				t.Fatalf("provision conflict = status %d, body %s", response.StatusCode, result)
			}
		})
	}
}

func TestManualConnectionCreateReturnsGone(t *testing.T) {
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	service := application.NewService(&testStore{records: map[string]application.StoredConnection{}},
		testCipher{}, testOwner{}, testEnvironment{}, failedProbe{})
	NewHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, continueRequest)(app)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/projects/p1/db",
		strings.NewReader(`{"type":"postgres"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(secureSessionCookie())
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusGone || !strings.Contains(string(body), "manual_connection_removed") {
		t.Fatalf("manual connection create response = %d %s", response.StatusCode, body)
	}
}

type busyStore struct{}

func (busyStore) Get(context.Context, string) (application.StoredConnection, error) {
	return application.StoredConnection{}, application.ProvisionError{Category: application.ProvisionInProgress}
}

func (busyStore) Save(context.Context, string, application.StoredConnection) error {
	return application.ProvisionError{Category: application.ProvisionInProgress}
}

func (busyStore) Delete(context.Context, string) error {
	return application.ProvisionError{Category: application.ProvisionInProgress}
}

var errSecretProvisionFailure = errors.New("failed for postgres://admin:admin-secret@db/project-secret")

type errorProvisioner struct{ err error }

func (errorProvisioner) Configured(context.Context) (bool, error) { return true, nil }
func (p errorProvisioner) Provision(context.Context, string) (application.ProvisionResult, error) {
	return application.ProvisionResult{}, p.err
}
func (errorProvisioner) Restore(context.Context, string, string) error { return nil }
