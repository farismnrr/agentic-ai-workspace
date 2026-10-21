package http

import (
	"net/http"
	"strings"
	"testing"
)

func TestDedicatedPreviewRequestForwardsAppCookieOnly(t *testing.T) {
	in := http.Header{}
	in.Set("Cookie", "session=workspace-secret; pukat_session=app-secret; "+previewAccessCookie+"=private-ticket")
	in.Set("Authorization", "Bearer private")
	out := forSiteRequest(in, "/run/pukat-app", "127.0.0.1", true)
	if out.Get("Cookie") != "pukat_session=app-secret" {
		t.Fatalf("unsafe cookies forwarded: %q", out.Get("Cookie"))
	}
	if out.Get("Authorization") != "" {
		t.Fatal("Workspace Authorization forwarded")
	}
	legacy := forSiteRequest(in, "/run/pukat-app", "127.0.0.1", false)
	if legacy.Get("Cookie") != "" {
		t.Fatal("legacy sandbox now forwards cookies")
	}
}

func TestDedicatedPreviewResponseSetsOnlyHostScopedAppCookies(t *testing.T) {
	in := http.Header{}
	in.Add("Set-Cookie", "pukat_session=ok; Path=/; HttpOnly; Secure; SameSite=Lax")
	in.Add("Set-Cookie", "session=bad; Path=/")
	in.Add("Set-Cookie", "pukat_session=bad; Domain=workspace.example; Path=/")
	in.Add("Set-Cookie", previewAccessCookie+"=bad; Path=/")
	in.Set("Content-Security-Policy", "default-src 'self'")
	in.Set("Access-Control-Allow-Origin", "*")
	out := forSiteResponse(in, "/run/pukat-app", "/run/pukat-app", true)
	if len(out.Values("Set-Cookie")) != 1 || !strings.HasPrefix(out.Get("Set-Cookie"), "pukat_session=ok;") {
		t.Fatalf("unsafe response cookies: %v", out.Values("Set-Cookie"))
	}
	if out.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("wildcard CORS on auth preview")
	}
	if strings.Contains(out.Get("Content-Security-Policy"), "sandbox") ||
		!strings.Contains(out.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("wrong preview CSP: %v", out.Get("Content-Security-Policy"))
	}
	legacy := forSiteResponse(in, "/run/pukat-app", "/run/pukat-app", false)
	if len(legacy.Values("Set-Cookie")) != 0 || !strings.Contains(legacy.Get("Content-Security-Policy"), "sandbox") {
		t.Fatal("legacy isolation weakened")
	}
}

func TestPreviewDomainMustMatchProject(t *testing.T) {
	h := NewHandler(nil, nil, false).WithPreviewDomain("preview.example.net")
	if !h.isPreviewHost("pukat-app.preview.example.net", "pukat-app") {
		t.Fatal("valid project host rejected")
	}
	for _, host := range []string{
		"evil.preview.example.net", "pukat-app.preview.example.net.evil.com", "workspace.example.net",
	} {
		if h.isPreviewHost(host, "pukat-app") {
			t.Fatalf("unexpected project host %q", host)
		}
	}
}
