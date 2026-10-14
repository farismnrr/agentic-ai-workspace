package cimd

import "testing"

func TestParseDocument(t *testing.T) {
	t.Parallel()

	const id = "https://chatgpt.com/oauth/client.json"

	good := `{"client_id":"` + id + `","client_name":"ChatGPT","redirect_uris":["https://chatgpt.com/cb"]}`

	client, err := parseDocument(id, []byte(good))
	if err != nil || client.Name != "ChatGPT" || !client.AllowsRedirect("https://chatgpt.com/cb") {
		t.Fatalf("valid document rejected: %+v %v", client, err)
	}

	for name, body := range map[string]string{
		"mismatched id": `{"client_id":"https://evil.test/c.json","redirect_uris":["https://chatgpt.com/cb"]}`,
		"no redirects":  `{"client_id":"` + id + `"}`,
		"bad redirect":  `{"client_id":"` + id + `","redirect_uris":["http://evil.test/cb"]}`,
		"not json":      `<html>`,
	} {
		if _, err := parseDocument(id, []byte(body)); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}
