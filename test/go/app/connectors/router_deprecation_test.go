package connectors_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestLegacy9routerCannotBeAddedThroughAPI(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie, _ := apptest.SignedInClient(t, env.App)
	for _, tc := range []struct{ name, url string }{
		{"9router", "http://localhost:9191/v1"},
		{"Local API", "http://localhost:20128/v1"},
		{"9Router account bridge", "http://127.0.0.1:9999/v1"},
	} {
		body := `{"kind":"compatible","name":"` + tc.name +
			`","baseUrl":"` + tc.url + `","apiKey":"test-key-not-secret"}`
		result := apptest.Do(t, env.App, "POST", "/api/providers", body, apptest.JSONCookie(cookie))
		if result.Status != 400 || !strings.Contains(string(result.Body), "account_relay_disabled") {
			t.Errorf("%s: got %d %s", tc.name, result.Status, result.Body)
		}
	}
}
