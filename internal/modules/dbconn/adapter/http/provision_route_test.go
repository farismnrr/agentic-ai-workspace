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

type routeProvisionerRegistry map[domain.Kind]application.Provisioner

func (r routeProvisionerRegistry) For(kind domain.Kind) application.Provisioner {
	return r[kind]
}

func TestProvisionRouteSelectsRequestedKind(t *testing.T) {
	store := &testStore{records: map[string]application.StoredConnection{}}
	service := application.NewServiceWithProvisioners(
		store, testCipher{}, testOwner{}, testEnvironment{}, failedProbe{},
		routeProvisionerRegistry{domain.MySQL: routeProvisioner{}},
	)
	app := fiber.New()
	continueRequest := func(c fiber.Ctx) error { return c.Next() }
	NewHandler(service).Mount(httpserver.RequireSession(testSessions{}), continueRequest, continueRequest)(app)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/api/projects/p1/db/provision", strings.NewReader(`{"kind":"mysql"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(secureSessionCookie())
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"choice":"mysql"`) ||
		!strings.Contains(string(body), `"type":"mysql"`) {
		t.Fatalf("MySQL provision response = %d %s", response.StatusCode, body)
	}
}
