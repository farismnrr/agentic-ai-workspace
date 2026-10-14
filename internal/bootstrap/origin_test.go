package bootstrap

import "testing"

func register(t *testing.T, origin, email string) int {
	t.Helper()

	body := `{"name":"Sari","email":"` + email + `","password":"correct horse"}`
	headers := map[string]string{"Content-Type": "application/json", "Origin": origin}

	return do(t, newApp(t), "POST", "/api/auth/register", body, headers).Status
}

func TestConfiguredAndPublicOriginsMayPost(t *testing.T) {
	t.Parallel()

	for _, origin := range []string{"http://localhost:8081", "http://127.0.0.1:8081", publicURL} {
		if got := register(t, origin, "a@example.com"); got != 201 {
			t.Errorf("origin %s must be allowed, got %d", origin, got)
		}
	}
}

func TestUnlistedOriginIsRejected(t *testing.T) {
	t.Parallel()

	for _, origin := range []string{"https://evil.test", "http://localhost:9999", "http://example.com"} {
		if got := register(t, origin, "b@example.com"); got != 403 {
			t.Errorf("origin %s must be rejected, got %d", origin, got)
		}
	}
}

func TestCORSFollowsConfiguredOrigins(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	const path = "/.well-known/oauth-authorization-server"

	allowed := do(t, app, "GET", path, "", map[string]string{"Origin": "http://localhost:8081"})
	foreign := do(t, app, "GET", path, "", map[string]string{"Origin": "https://evil.test"})

	if allowed.Header.Get("Access-Control-Allow-Origin") != "http://localhost:8081" {
		t.Errorf("listed origin must be echoed, got %q", allowed.Header.Get("Access-Control-Allow-Origin"))
	}

	if foreign.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("foreign origin must not get CORS headers, got %q", foreign.Header.Get("Access-Control-Allow-Origin"))
	}
}
