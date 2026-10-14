package http

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

const (
	siteRoot   = "/run/sensio/tok/"
	ticketRoot = "/run/sensio/tok/~" + goodTicket + "/"
)

// as returns the headers a browser sends for what it is doing: dest is "document" for a page
// the person opens, or "script", "empty" (fetch) and so on for what a page loads itself.
func as(dest, cookie string) map[string]string {
	headers := map[string]string{"Sec-Fetch-Dest": dest, "Sec-Fetch-Mode": "no-cors"}
	if dest == "document" {
		headers["Sec-Fetch-Mode"] = "navigate"
	}

	if cookie != "" {
		headers["Cookie"] = "session=" + cookie
	}

	return headers
}

func TestPrivatePagesNeedTheOwnerAndAskOthersToSignIn(t *testing.T) {
	app, _ := newApp(t, domain.Private, domain.Running)

	cases := map[string]struct {
		headers map[string]string
		want    int
	}{
		"anonymous":                {as("document", ""), 403},
		"someone else":             {as("document", "other-session"), 403},
		"older browser, anonymous": {nil, 403},
	}

	for name, c := range cases {
		status, header, body := (call{path: ticketRoot + "echo", headers: c.headers}).do(t, app)
		sandboxed := strings.HasPrefix(header.Get("Content-Security-Policy"), "sandbox")
		if status != c.want || !strings.Contains(body, "/login?next=") || !sandboxed {
			t.Errorf("%s: status %d, body %q", name, status, body)
		}
	}
}

func TestPrivateSiteIgnoresForgedHeadersAndAddressAlone(t *testing.T) {
	app, seen := newApp(t, domain.Private, domain.Running)

	// curl can send any Sec-Fetch-* header, and the uuid in the address is all an outsider may know.
	for name, headers := range map[string]map[string]string{
		"claims to be a script": as("script", ""),
		"claims to be fetch":    as("empty", ""),
		"someone else's login":  as("script", "other-session"),
		"no metadata":           nil,
	} {
		if got, _, _ := (call{path: siteRoot + "echo", headers: headers}).do(t, app); got != 403 {
			t.Errorf("%s: status %d, want 403", name, got)
		}
	}

	forged := "/run/sensio/tok/~9999999999.forged/echo"
	if got, _, _ := (call{path: forged, headers: as("script", "")}).do(t, app); got != 403 {
		t.Errorf("a forged ticket: status %d, want 403", got)
	}

	if len(seen) != 0 {
		t.Error("nothing may reach the app")
	}
}

func TestOwnerOpensPrivateSiteThroughATicketedAddress(t *testing.T) {
	app, seen := newApp(t, domain.Private, domain.Running)
	owner := as("document", "owner-session")

	status, header, _ := (call{path: siteRoot + "echo?x=1", headers: owner}).do(t, app)
	if status != http.StatusFound || header.Get("Location") != "/run/sensio/tok/~"+goodTicket+"/echo?x=1" {
		t.Fatalf("the owner must be sent to a ticketed address, got %d %q", status, header.Get("Location"))
	}

	status, _, body := (call{path: header.Get("Location"), headers: owner}).do(t, app)
	if status != 200 || !strings.Contains(body, `"path":"/echo"`) || !strings.Contains(body, `"query":"x=1"`) {
		t.Fatalf("status %d body %s", status, body)
	}

	if got := <-seen; got.Get("X-Forwarded-Prefix") != "/run/sensio/tok/~"+goodTicket || got.Get("Cookie") != "" {
		t.Errorf("the app sees the ticketed prefix and no login: %v", got)
	}
}

func TestATicketServesWhatThePageLoadsButNeverThePageItself(t *testing.T) {
	app, _ := newApp(t, domain.Private, domain.Running)

	// What the page loads itself carries the ticket in the address and no login.
	if got, _, _ := (call{path: ticketRoot + "echo", headers: as("script", "")}).do(t, app); got != 200 {
		t.Errorf("a sub-request with a ticket: status %d", got)
	}

	// But a page never opens on a ticket alone, so a copied address is useless without the login.
	if got, _, _ := (call{path: ticketRoot + "echo", headers: as("document", "")}).do(t, app); got != 403 {
		t.Errorf("a page with a ticket but no login: status %d, want 403", got)
	}

	// A form post from the page has no login either (cookies are not sent), so the ticket serves.
	post := call{method: http.MethodPost, path: ticketRoot + "echo", headers: map[string]string{
		"Sec-Fetch-Dest": "document", "Sec-Fetch-Mode": "navigate",
	}}
	if got, _, _ := post.do(t, app); got != 200 {
		t.Errorf("a form post with a ticket: status %d", got)
	}
}

func TestPublicSiteOpensForAnyoneWithoutATicket(t *testing.T) {
	app, _ := newApp(t, domain.Public, domain.Running)

	if got, _, _ := (call{path: siteRoot + "echo", headers: as("document", "")}).do(t, app); got != 200 {
		t.Fatalf("status %d", got)
	}

	// An app path that starts with ~ is the app's own on a public site, so it reaches the app (which has no such page).
	if got, _, _ := (call{path: siteRoot + "~x/echo", headers: as("document", "")}).do(t, app); got != 404 {
		t.Fatalf("status %d, want the app's own 404", got)
	}
}

func TestUnknownAndStoppedSitesAreNotFoundAndNeverFallIntoTheApp(t *testing.T) {
	stopped, _ := newApp(t, domain.Public, domain.Stopped)
	running, _ := newApp(t, domain.Public, domain.Running)

	paths := map[string]struct {
		app  *fiber.App
		path string
	}{
		"stopped":       {stopped, "/run/sensio/tok/echo"},
		"unknown token": {running, "/run/sensio/nope/echo"},
	}

	for name, c := range paths {
		status, header, body := (call{path: c.path, headers: as("document", "owner-session")}).do(t, c.app)
		if status != 404 || strings.Contains(body, "SPA") {
			t.Errorf("%s: status %d body %q", name, status, body)
		}

		if !strings.HasPrefix(header.Get("Content-Security-Policy"), "sandbox") {
			t.Errorf("%s: even errors must be sandboxed", name)
		}
	}
}
