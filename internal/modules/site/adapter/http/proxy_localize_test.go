package http

import (
	"strings"
	"testing"
)

func TestProxyPutsTheAddressBackOnRootLinks(t *testing.T) {
	app, _ := newApp(t, "public", "running")

	status, header, body := (call{path: "/run/sensio/evil", headers: as("document", "")}).do(t, app)
	if status != 200 || !strings.Contains(header.Get("Content-Type"), "html") {
		t.Fatalf("status %d type %q", status, header.Get("Content-Type"))
	}

	if !strings.Contains(body, `<script data-site-shim>`) || !strings.Contains(body, `var P="/run/sensio"`) {
		t.Fatalf("the page must carry the shim for its address:\n%s", body)
	}
}
