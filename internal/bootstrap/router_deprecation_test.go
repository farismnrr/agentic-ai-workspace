package bootstrap

import (
	"strings"
	"testing"
)

func TestLegacy9routerCannotBeAddedThroughAPI(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)
	for _, tc := range []struct{ name, url string }{
		{"9router", "http://localhost:9191/v1"},
		{"Local API", "http://localhost:20128/v1"},
		{"9Router account bridge", "http://127.0.0.1:9999/v1"},
	} {
		body := `{"kind":"compatible","name":"` + tc.name +
			`","baseUrl":"` + tc.url + `","apiKey":"test-key-not-secret"}`
		result := do(t, env.app, "POST", "/api/providers", body, jsonCookie(cookie))
		if result.Status != 400 || !strings.Contains(string(result.Body), "account_relay_disabled") {
			t.Errorf("%s: got %d %s", tc.name, result.Status, result.Body)
		}
	}
}
