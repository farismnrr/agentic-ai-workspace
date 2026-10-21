package http

import (
	"net/http"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// passFrom reads the visitor pass a page response handed to the browser.
func passFrom(t *testing.T, header http.Header) string {
	t.Helper()

	for _, cookie := range header.Values("Set-Cookie") {
		if value, found := strings.CutPrefix(cookie, visitorCookie+"="); found {
			pass, _, _ := strings.Cut(value, ";")

			return pass
		}
	}

	t.Fatal("a page on a public site must hand the browser a pass")

	return ""
}

func withPass(pass string) map[string]string {
	headers := as("empty", "")
	headers[visitorHeader] = pass

	return headers
}

func TestPublicPageCarriesAPassInACookieOfItsOwnAddress(t *testing.T) {
	app, _ := newApp(t, domain.Public, domain.Running)

	_, header, body := (call{path: siteRoot + "evil", headers: as("document", "")}).do(t, app)
	pass := passFrom(t, header)

	if !strings.Contains(body, `window.__SITE_VISITOR__="`+pass+`"`) {
		t.Error("the page must carry the pass for the shim to send")
	}

	for _, cookie := range header.Values("Set-Cookie") {
		lower := strings.ToLower(cookie)
		if !strings.Contains(lower, "httponly") || !strings.Contains(lower, "path=/run/sensio/") ||
			!strings.Contains(lower, "samesite=lax") || !strings.Contains(lower, "secure") {
			t.Errorf("the pass cookie must be private to the site's address: %q", cookie)
		}
	}
}

func TestPublicVisitorKeepsTheAppLoginAcrossRequestsAndPages(t *testing.T) {
	app, _ := newApp(t, domain.Public, domain.Running)

	_, header, _ := (call{path: siteRoot + "evil", headers: as("document", "")}).do(t, app)
	pass := passFrom(t, header)

	_, header, _ = (call{path: siteRoot + "login", headers: withPass(pass)}).do(t, app)
	if header.Get("Set-Cookie") != "" {
		t.Errorf("the app's cookie must not reach the browser: %q", header.Get("Set-Cookie"))
	}

	if _, _, got := (call{path: siteRoot + "me", headers: withPass(pass)}).do(t, app); !strings.Contains(got, "sid=abc") {
		t.Errorf("the next request carries the login: %q", got)
	}

	// The next page the person opens sends the cookie, and it is the same visitor.
	next := as("document", "")
	next["Cookie"] = visitorCookie + "=" + pass
	_, header, _ = (call{path: siteRoot + "evil", headers: next}).do(t, app)
	again := passFrom(t, header)

	if strings.Split(again, ".")[0] != strings.Split(pass, ".")[0] {
		t.Errorf("the same browser keeps the same visitor: %q then %q", pass, again)
	}

	if _, _, got := (call{path: siteRoot + "me", headers: withPass(again)}).do(t, app); !strings.Contains(got, "sid=abc") {
		t.Errorf("the login survives a new page: %q", got)
	}
}

func TestPublicVisitorsAreKeptApartAndPassesCannotBeForged(t *testing.T) {
	app, seen := newApp(t, domain.Public, domain.Running)

	_, first, _ := (call{path: siteRoot + "evil", headers: as("document", "")}).do(t, app)
	_, second, _ := (call{path: siteRoot + "evil", headers: as("document", "")}).do(t, app)
	pass, other := passFrom(t, first), passFrom(t, second)

	_, _, _ = (call{path: siteRoot + "login", headers: withPass(pass)}).do(t, app)

	forged := strings.Replace(pass, "9999999999", "9999999998", 1)
	for name, headers := range map[string]map[string]string{
		"another browser":       withPass(other),
		"a forged pass":         withPass(forged),
		"no pass":               as("empty", ""),
		"a cookie, not a page":  {"Sec-Fetch-Dest": "empty", "Sec-Fetch-Mode": "cors", "Cookie": visitorCookie + "=" + pass},
		"a pass for a bad site": withPass("zz." + pass),
	} {
		if _, _, got := (call{path: siteRoot + "me", headers: headers}).do(t, app); strings.Contains(got, "sid=abc") {
			t.Errorf("%s must not get this visitor's login", name)
		}
	}

	// The app never sees the pass.
	_, _, _ = (call{path: siteRoot + "echo", headers: withPass(pass)}).do(t, app)

	if got := <-seen; got.Get(visitorHeader) != "" || strings.Contains(got.Get("Cookie"), visitorCookie) {
		t.Errorf("the pass must not reach the app: %v", got)
	}
}

func TestPrivateAndPreviewSitesAreNotGivenAPass(t *testing.T) {
	app, _ := newApp(t, domain.Private, domain.Running)

	_, header, _ := (call{path: ticketRoot + "echo", headers: as("script", "")}).do(t, app)
	if len(header.Values("Set-Cookie")) != 0 {
		t.Errorf("a private site knows its visitor by the ticket: %v", header.Values("Set-Cookie"))
	}
}
