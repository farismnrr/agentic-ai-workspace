package bootstrap

import (
	"os"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

const siteCommand = `echo '<h1>hello site</h1>' > index.html && exec python3 -m http.server $PORT --bind 127.0.0.1`

// TestAnAppTheAIStartsCanBeOpenedPublishedAndStopped runs the whole path with a real container:
// the model asks to start an app, the person allows it, the app is opened through /run, and
// the owner publishes and stops it. It needs the toolbox image (SANDBOX_TEST_IMAGE).
func TestAnAppTheAIStartsCanBeOpenedPublishedAndStopped(t *testing.T) {
	t.Parallel()

	image := os.Getenv("SANDBOX_TEST_IMAGE")
	if image == "" {
		t.Skip("set SANDBOX_TEST_IMAGE to the toolbox image to run this test")
	}

	a := newAgentRigWith(t, func(c *config.Config) { c.SandboxImage = image },
		toolCall("s1", "start_site", map[string]any{"command": siteCommand, "port": 8123}),
		finishCall("Your page is running"),
	)

	id := a.start(t, "Show me a page")
	a.waitApproving(t, id, "done")

	var url string

	for _, event := range a.thread(t, id) {
		if event.Kind == "site" {
			url = event.URL
		}
	}

	if !strings.HasPrefix(url, "/run/bakery/") || !strings.HasSuffix(url, "/") {
		t.Fatalf("the thread must offer a link under /run, got %q", url)
	}

	owner := map[string]string{"Cookie": a.cookie, "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}
	anonymous := map[string]string{"Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}

	expectSandboxedPage(t, a, url, owner)

	if got := do(t, a.env.app, "GET", url, "", anonymous).Status; got != 404 {
		t.Fatalf("a private site must not open for an anonymous visitor, got %d", got)
	}

	listed := do(t, a.env.app, "GET", "/api/sites", "", owner).Body
	mustContain(t, listed, `"status":"running"`, `"visibility":"private"`, url)

	a.expectStatus(t, "PATCH", "/api/sites/"+a.project, `{"visibility":"public"}`, 204)
	mustContain(t, do(t, a.env.app, "GET", url, "", anonymous).Body, "hello site")

	a.expectStatus(t, "POST", "/api/sites/"+a.project+"/stop", "", 204)

	if got := do(t, a.env.app, "GET", url, "", owner).Status; got != 404 {
		t.Fatalf("a stopped site must be gone, got %d", got)
	}
}

func expectSandboxedPage(t *testing.T, a agentRig, url string, headers map[string]string) {
	t.Helper()

	page := do(t, a.env.app, "GET", url, "", headers)
	mustContain(t, page.Body, "hello site")

	csp := page.Header.Get("Content-Security-Policy")
	if !strings.HasPrefix(csp, "sandbox ") || strings.Contains(csp, "allow-same-origin") {
		t.Fatalf("the page must be sandboxed, got %q", csp)
	}
}

func (a agentRig) expectStatus(t *testing.T, method, path, body string, want int) {
	t.Helper()

	if got := do(t, a.env.app, method, path, body, jsonCookie(a.cookie)).Status; got != want {
		t.Fatalf("%s %s: %d, want %d", method, path, got, want)
	}
}
