package sites_test

import (
	"os"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
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

	a := apptest.NewAgentRigWith(t, func(c *config.Config) { c.SandboxImage = image },
		apptest.ToolCall("s1", "start_site", map[string]any{"command": siteCommand, "port": 8123}),
		apptest.FinishCall("Your page is running"),
	)

	id := a.Start(t, "Show me a page")
	a.WaitApproving(t, id, "done")

	var url string

	for _, event := range a.Thread(t, id) {
		if event.Kind == "site" {
			url = event.URL
		}
	}

	if !strings.HasPrefix(url, "/run/bakery/") || !strings.HasSuffix(url, "/") {
		t.Fatalf("the thread must offer a link under /run, got %q", url)
	}

	page := map[string]string{"Cookie": a.Cookie, "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}
	anonymous := map[string]string{"Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}
	script := map[string]string{"Sec-Fetch-Mode": "no-cors", "Sec-Fetch-Dest": "script"}

	ticketed := expectOwnerOpens(t, a, url, page)

	// Private means private: neither a stranger's page nor a script request that merely knows the address.
	// Anonymous navigation may redirect to the app's forbidden page, but
	// must never receive private site content or a ticketed site URL.
	denied := apptest.Do(t, a.Environment.App, "GET", url, "", anonymous)
	if denied.Status != 302 || !strings.HasPrefix(denied.Header.Get("Location"), "/forbidden?next=") {
		t.Fatalf("anonymous navigation must lead to the forbidden page, got %d %q",
			denied.Status, denied.Header.Get("Location"))
	}
	if got := apptest.Do(t, a.Environment.App, "GET", url, "", script).Status; got != 403 {
		t.Fatalf("anonymous script request to a private site must be forbidden, got %d", got)
	}

	// What the page loads itself carries the ticket the owner's visit put in the address.
	apptest.MustContain(t, apptest.Do(t, a.Environment.App, "GET", ticketed, "", script).Body, "hello site")

	listed := apptest.Do(t, a.Environment.App, "GET", "/api/sites", "", page).Body
	apptest.MustContain(t, listed, `"status":"running"`, `"visibility":"private"`, url)

	expectStatus(t, a, "PATCH", "/api/sites/"+a.Project, `{"visibility":"public"}`, 204)
	apptest.MustContain(t, apptest.Do(t, a.Environment.App, "GET", url, "", anonymous).Body, "hello site")

	expectStatus(t, a, "POST", "/api/sites/"+a.Project+"/stop", "", 204)

	if got := apptest.Do(t, a.Environment.App, "GET", url, "", page).Status; got != 404 {
		t.Fatalf("a stopped site must be gone, got %d", got)
	}
}

// expectOwnerOpens follows the owner's visit: a redirect to a ticketed address, then a sandboxed page.
func expectOwnerOpens(t *testing.T, a apptest.AgentRig, url string, headers map[string]string) string {
	t.Helper()

	redirect := apptest.Do(t, a.Environment.App, "GET", url, "", headers)
	ticketed := redirect.Header.Get("Location")

	if redirect.Status != 302 || !strings.Contains(ticketed, "/~") {
		t.Fatalf("the owner must be sent to a ticketed address, got %d %q", redirect.Status, ticketed)
	}

	expectSandboxedPage(t, a, ticketed, headers)

	return ticketed
}

func expectSandboxedPage(t *testing.T, a apptest.AgentRig, url string, headers map[string]string) {
	t.Helper()

	page := apptest.Do(t, a.Environment.App, "GET", url, "", headers)
	apptest.MustContain(t, page.Body, "hello site")

	csp := page.Header.Get("Content-Security-Policy")
	if !strings.HasPrefix(csp, "sandbox ") || strings.Contains(csp, "allow-same-origin") {
		t.Fatalf("the page must be sandboxed, got %q", csp)
	}
}

func expectStatus(t *testing.T, a apptest.AgentRig, method, path, body string, want int) {
	t.Helper()

	if got := apptest.Do(t, a.Environment.App, method, path, body, apptest.JSONCookie(a.Cookie)).Status; got != want {
		t.Fatalf("%s %s: %d, want %d", method, path, got, want)
	}
}
