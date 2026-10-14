package http

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func TestPrivateSiteOpensOnlyForItsOwner(t *testing.T) {
	app, _ := newApp(t, domain.Private, domain.Running)

	cases := map[string]struct {
		headers map[string]string
		want    int
	}{
		"owner page":                       {as("document", "owner-session"), 200},
		"anonymous page":                   {as("document", ""), 404},
		"someone else's page":              {as("document", "other-session"), 404},
		"the page's own script (no login)": {as("script", ""), 200},
		"older browser, anonymous":         {nil, 404},
	}

	for name, c := range cases {
		if got, _, _ := (call{path: "/run/sensio/tok/echo", headers: c.headers}).do(t, app); got != c.want {
			t.Errorf("%s: status %d, want %d", name, got, c.want)
		}
	}
}

func TestPublicSiteOpensForAnyone(t *testing.T) {
	app, _ := newApp(t, domain.Public, domain.Running)

	if got, _, _ := (call{path: "/run/sensio/tok/echo", headers: as("document", "")}).do(t, app); got != 200 {
		t.Fatalf("status %d", got)
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
