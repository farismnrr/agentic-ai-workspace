package bootstrap

import (
	"strings"
	"testing"
)

func TestSitesAreMountedAndNeverFallIntoTheApp(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	page := map[string]string{"Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}
	res := do(t, app, "GET", "/run/demo/00000000-0000-4000-8000-000000000000/", "", page)

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

	app := newApp(t)

	if res := do(t, app, "GET", "/api/sites", "", nil); res.Status != 401 {
		t.Fatalf("signed out: %d", res.Status)
	}

	cookie, _ := signedInClient(t, app)
	mine := map[string]string{"Cookie": "session=" + strings.TrimPrefix(cookie, "session=")}

	res := do(t, app, "GET", "/api/sites", "", mine)
	if res.Status != 200 || !strings.Contains(string(res.Body), `"sites":[]`) {
		t.Fatalf("signed in: %d %s", res.Status, res.Body)
	}

	sandboxed := map[string]string{
		"Cookie": mine["Cookie"], "Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "cors", "Sec-Fetch-Dest": "empty",
	}
	if res = do(t, app, "GET", "/api/sites", "", sandboxed); res.Status != 403 {
		t.Fatalf("a sandboxed page must not use the session, got %d", res.Status)
	}
}
