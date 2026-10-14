package cimd

import "testing"

func TestCheckAddressBlocksNonPublic(t *testing.T) {
	t.Parallel()

	for _, address := range []string{
		"127.0.0.1:443", "[::1]:443", "10.0.0.5:443", "192.168.1.1:443", "172.16.0.1:443",
		"169.254.169.254:80", "100.64.0.1:443", "0.0.0.0:80", "[::ffff:10.0.0.1]:443", "[fe80::1]:443",
	} {
		if err := checkAddress(address); err == nil {
			t.Errorf("%s must be blocked", address)
		}
	}

	if err := checkAddress("93.184.216.34:443"); err != nil {
		t.Errorf("public address must pass: %v", err)
	}
}

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
