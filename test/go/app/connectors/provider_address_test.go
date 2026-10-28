package connectors_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func providerBody(kind, address string) string {
	body := `{"kind":"` + kind + `","name":"x","apiKey":"` + "k" + `"`
	if address != "" {
		body += `,"baseUrl":"` + address + `"`
	}

	return body + `}`
}

func TestProviderAddressesCannotAimTheServerElsewhere(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie, _ := apptest.SignedInClient(t, env.App)

	refused := []struct{ kind, address, code string }{
		{"compatible", "http://192.168.1.5:20128/v1", "invalid_base_url"},
		{"compatible", "http://169.254.169.254/latest", "invalid_base_url"},
		{"compatible", "https://evil.example.com/v1", "invalid_base_url"},
		{"openai", "http://localhost:8081", "address_fixed"},
		{"gemini", "https://evil.example.com", "address_fixed"},
	}

	for _, c := range refused {
		res := apptest.Do(t, env.App, "POST", "/api/providers", providerBody(c.kind, c.address), apptest.JSONCookie(cookie))
		if res.Status != 400 {
			t.Errorf("%s %s must be refused, got %d", c.kind, c.address, res.Status)
		}

		apptest.MustContain(t, res.Body, c.code)
	}

	// A hosted provider without an address gets its own, and a local gateway is fine.
	hosted := apptest.Do(t, env.App, "POST", "/api/providers", providerBody("gemini", ""), apptest.JSONCookie(cookie))
	apptest.MustContain(t, hosted.Body, "https://generativelanguage.googleapis.com/v1beta/openai")

	local := providerBody("compatible", "http://localhost:9010/v1")
	if got := apptest.Do(t, env.App, "POST", "/api/providers", local, apptest.JSONCookie(cookie)).Status; got != 201 {
		t.Errorf("a gateway on this computer must be accepted, got %d", got)
	}
}
