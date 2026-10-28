package egressgrant_test

import (
	"net/http"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestEgressAPIDoesNotExistWhenFlagOff(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie := apptest.Register(t, env.App, "A", "a@example.test", "a sufficiently long password")
	for _, path := range []string{
		"/api/projects/p1/egress-grants/", "/api/projects/p1/egress-grants/csrf",
		"/api/projects/p1/egress-grants/g1/approve",
	} {
		response := apptest.Do(t, env.App, http.MethodGet, path, "", apptest.JSONCookie(cookie))
		if response.Status != http.StatusNotFound {
			t.Errorf("%s status %d", path, response.Status)
		}
	}
}
