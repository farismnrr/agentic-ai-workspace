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

	page := map[string]string{"Cookie": a.cookie, "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}
	anonymous := map[string]string{"Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}
	script := map[string]string{"Sec-Fetch-Mode": "no-cors", "Sec-Fetch-Dest": "script"}

	ticketed := expectOwnerOpens(t, a, url, page)

	// Private means private: neither a stranger's page nor a script request that merely knows the address.
	for name, headers := range map[string]map[string]string{"page": anonymous, "script": script} {
		if got := do(t, a.env.app, "GET", url, "", headers).Status; got != 403 {
			t.Fatalf("an anonymous %s request to a private site must be forbidden, got %d", name, got)
		}
	}

	// What the page loads itself carries the ticket the owner's visit put in the address.
	mustContain(t, do(t, a.env.app, "GET", ticketed, "", script).Body, "hello site")

	listed := do(t, a.env.app, "GET", "/api/sites", "", page).Body
	mustContain(t, listed, `"status":"running"`, `"visibility":"private"`, url)

	a.expectStatus(t, "PATCH", "/api/sites/"+a.project, `{"visibility":"public"}`, 204)
	mustContain(t, do(t, a.env.app, "GET", url, "", anonymous).Body, "hello site")

	a.expectStatus(t, "POST", "/api/sites/"+a.project+"/stop", "", 204)

	if got := do(t, a.env.app, "GET", url, "", page).Status; got != 404 {
		t.Fatalf("a stopped site must be gone, got %d", got)
	}
}

// expectOwnerOpens follows the owner's visit: a redirect to a ticketed address, then a sandboxed page.
func expectOwnerOpens(t *testing.T, a agentRig, url string, headers map[string]string) string {
	t.Helper()

	redirect := do(t, a.env.app, "GET", url, "", headers)
	ticketed := redirect.Header.Get("Location")

	if redirect.Status != 302 || !strings.Contains(ticketed, "/~") {
		t.Fatalf("the owner must be sent to a ticketed address, got %d %q", redirect.Status, ticketed)
	}

	expectSandboxedPage(t, a, ticketed, headers)

	return ticketed
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
