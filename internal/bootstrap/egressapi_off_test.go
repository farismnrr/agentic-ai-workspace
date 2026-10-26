package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// Egress approval endpoints must not even exist when the feature is disabled.
func TestEgressAPIDoesNotExistWhenFlagOff(t *testing.T) {
	mount, err := newEgressAPIMount(nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	mount(app)
	for _, path := range []string{
		"/api/projects/p1/egress-grants/", "/api/projects/p1/egress-grants/csrf",
		"/api/projects/p1/egress-grants/g1/approve",
	} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		response, testErr := app.Test(req)
		if testErr != nil {
			t.Fatal(testErr)
		}
		if response.StatusCode != http.StatusNotFound {
			t.Errorf("%s status %d", path, response.StatusCode)
		}
		_ = response.Body.Close()
	}
}
