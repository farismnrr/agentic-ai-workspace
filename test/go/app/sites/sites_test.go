package sites_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestSitesAreMountedAndNeverFallIntoTheApp(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)

	page := map[string]string{"Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}
	res := apptest.Do(t, env.App, "GET", "/run/demo/00000000-0000-4000-8000-000000000000/", "", page)

	sentAway := res.Status == 302 && res.Header.Get("Location") == "/not-found"
	if !sentAway || strings.Contains(strings.ToLower(string(res.Body)), "<html") {
		t.Fatalf("an unknown site's page must lead to the not-found screen, got %d %s", res.Status, res.Body)
	}

	if csp := res.Header.Get("Content-Security-Policy"); !strings.HasPrefix(csp, "sandbox") {
		t.Fatalf("everything under /run is sandboxed, got %q", csp)
	}
}

func TestSitesAPINeedsASessionAndStartsEmpty(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)

	if res := apptest.Do(t, env.App, "GET", "/api/sites", "", nil); res.Status != 401 {
		t.Fatalf("signed out: %d", res.Status)
	}

	cookie, _ := apptest.SignedInClient(t, env.App)
	mine := map[string]string{"Cookie": "session=" + strings.TrimPrefix(cookie, "session=")}

	res := apptest.Do(t, env.App, "GET", "/api/sites", "", mine)
	if res.Status != 200 || !strings.Contains(string(res.Body), `"sites":[]`) {
		t.Fatalf("signed in: %d %s", res.Status, res.Body)
	}

	sandboxed := map[string]string{
		"Cookie": mine["Cookie"], "Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "cors", "Sec-Fetch-Dest": "empty",
	}
	if res = apptest.Do(t, env.App, "GET", "/api/sites", "", sandboxed); res.Status != 403 {
		t.Fatalf("a sandboxed page must not use the session, got %d", res.Status)
	}
}
