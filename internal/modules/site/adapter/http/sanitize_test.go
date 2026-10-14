package http

import (
	"net/http"
	"strings"
	"testing"
)

func TestRequestNeverCarriesTheLoginOrClientNames(t *testing.T) {
	in := http.Header{
		"Cookie":              {"session=secret"},
		"Authorization":       {"Bearer x"},
		"Service-Worker":      {"script"},
		"X-Forwarded-For":     {"6.6.6.6"},
		"Proxy-Authorization": {"Basic y"},
		"Proxy-Foo":           {"1"},
		"Accept":              {"text/html"},
		"Connection":          {"X-Custom, keep-alive"},
		"X-Custom":            {"hop"},
		"Upgrade":             {"websocket"},
	}

	out := forRequest(in, "/run/a/b", "1.2.3.4")

	dropped := []string{
		"Cookie", "Authorization", "Service-Worker", "Proxy-Authorization", "Proxy-Foo", "X-Custom", "Upgrade", "Connection",
	}

	for _, name := range dropped {
		if out.Get(name) != "" {
			t.Errorf("%s must be dropped, got %q", name, out.Get(name))
		}
	}

	if out.Get("Accept") != "text/html" || out.Get("X-Forwarded-Prefix") != "/run/a/b" ||
		out.Get("X-Forwarded-For") != "1.2.3.4" {
		t.Errorf("unexpected headers: %v", out)
	}

	if in.Get("Cookie") == "" {
		t.Error("the incoming header must not be modified")
	}
}

func TestConnectionCannotMakeTheProxyDropItsOwnHeaders(t *testing.T) {
	out := forRequest(http.Header{"Connection": {"X-Forwarded-Prefix, X-Forwarded-For"}}, "/p", "9.9.9.9")
	if out.Get("X-Forwarded-Prefix") != "/p" || out.Get("X-Forwarded-For") != "9.9.9.9" {
		t.Fatalf("our headers must survive a hostile Connection header: %v", out)
	}
}

func TestResponseIsSandboxedAndCannotTouchTheOrigin(t *testing.T) {
	in := http.Header{
		"Set-Cookie":                       {"session=evil; Path=/"},
		"Clear-Site-Data":                  {`"cookies"`},
		"Service-Worker-Allowed":           {"/"},
		"Strict-Transport-Security":        {"max-age=0"},
		"Content-Security-Policy":          {"default-src *"},
		"Access-Control-Allow-Origin":      {"https://evil.test"},
		"Access-Control-Allow-Credentials": {"true"},
		"Content-Type":                     {"text/html"},
		"Cache-Control":                    {"public, max-age=999"},
		"Location":                         {"/login"},
	}

	out := forResponse(in, "/run/a/b")

	forbidden := []string{
		"Set-Cookie", "Clear-Site-Data", "Service-Worker-Allowed", "Strict-Transport-Security",
		"Access-Control-Allow-Credentials",
	}

	for _, name := range forbidden {
		if out.Get(name) != "" {
			t.Errorf("%s must be dropped", name)
		}
	}

	csp := out.Get("Content-Security-Policy")
	if !strings.HasPrefix(csp, "sandbox ") || strings.Contains(csp, "allow-same-origin") ||
		strings.Contains(csp, "default-src") {
		t.Errorf("the sandbox policy must replace the app's: %q", csp)
	}

	want := map[string]string{
		"Access-Control-Allow-Origin": "*", "X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer",
		"Cache-Control": "no-store", "Content-Type": "text/html", "Location": "/run/a/b/login",
	}
	for name, value := range want {
		if out.Get(name) != value {
			t.Errorf("%s = %q, want %q", name, out.Get(name), value)
		}
	}
}

func TestOnlyRootRelativeRedirectsAreRewritten(t *testing.T) {
	for in, want := range map[string]string{
		"/x": "/run/a/b/x", "//evil.test/x": "//evil.test/x", "https://other.test/": "https://other.test/", "next": "next",
	} {
		if got := forResponse(http.Header{"Location": {in}}, "/run/a/b").Get("Location"); got != want {
			t.Errorf("Location %q -> %q, want %q", in, got, want)
		}
	}
}
