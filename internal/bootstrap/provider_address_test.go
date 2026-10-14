package bootstrap

import "testing"

func providerBody(kind, address string) string {
	body := `{"kind":"` + kind + `","name":"x","apiKey":"` + "k" + `"`
	if address != "" {
		body += `,"baseUrl":"` + address + `"`
	}

	return body + `}`
}

func TestProviderAddressesCannotAimTheServerElsewhere(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)

	refused := []struct{ kind, address, code string }{
		{"compatible", "http://192.168.1.5:20128/v1", "invalid_base_url"},
		{"compatible", "http://169.254.169.254/latest", "invalid_base_url"},
		{"compatible", "https://evil.example.com/v1", "invalid_base_url"},
		{"openai", "http://localhost:8081", "address_fixed"},
		{"gemini", "https://evil.example.com", "address_fixed"},
	}

	for _, c := range refused {
		res := do(t, env.app, "POST", "/api/providers", providerBody(c.kind, c.address), jsonCookie(cookie))
		if res.Status != 400 {
			t.Errorf("%s %s must be refused, got %d", c.kind, c.address, res.Status)
		}

		mustContain(t, res.Body, c.code)
	}

	// A hosted provider without an address gets its own, and a local gateway is fine.
	hosted := do(t, env.app, "POST", "/api/providers", providerBody("gemini", ""), jsonCookie(cookie))
	mustContain(t, hosted.Body, "https://generativelanguage.googleapis.com/v1beta/openai")

	local := providerBody("compatible", "http://localhost:20128/v1")
	if got := do(t, env.app, "POST", "/api/providers", local, jsonCookie(cookie)).Status; got != 201 {
		t.Errorf("a gateway on this computer must be accepted, got %d", got)
	}
}
