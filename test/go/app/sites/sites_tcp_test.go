package sites_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

// browser sends one request the way a browser does and never follows redirects, so each hop shows.
func browser(t *testing.T, base, path string, headers map[string]string) (int, http.Header, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, base+path, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	for name, value := range headers {
		req.Header.Set(name, value)
	}

	client := &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = res.Body.Close() }()

	body, _ := io.ReadAll(res.Body)

	return res.StatusCode, res.Header, string(body)
}

// startedSite runs the model's start_site call and returns the rig and the address in the thread.
func startedSite(t *testing.T, port int) (apptest.AgentRig, string) {
	t.Helper()

	image := os.Getenv("SANDBOX_TEST_IMAGE")
	if image == "" {
		t.Skip("set SANDBOX_TEST_IMAGE to the toolbox image to run this test")
	}

	a := apptest.NewAgentRigWith(t, func(c *config.Config) { c.SandboxImage = image },
		apptest.ToolCall("s1", "start_site", map[string]any{"command": apptest.SiteCommand, "port": port}),
		apptest.FinishCall("Your page is running"),
	)

	id := a.Start(t, "Show me a page")
	a.WaitApproving(t, id, "done")

	for _, event := range a.Thread(t, id) {
		if event.Kind == "site" {
			return a, event.URL
		}
	}

	t.Fatal("the thread has no site link")

	return a, ""
}

// listen serves the app on a real TCP port and returns its base address.
func listen(t *testing.T, a apptest.AgentRig) string {
	t.Helper()

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() { _ = a.Environment.App.Listener(ln) }()

	t.Cleanup(func() { _ = a.Environment.App.Shutdown() })

	return "http://" + ln.Addr().String()
}

// TestPrivateSiteOverRealHTTP repeats the private-site rules over a real TCP connection with a
// real container, as a browser would meet them: a signed-out visitor is forbidden whatever
// headers they claim, and the owner gets in through a ticketed address.
func TestPrivateSiteOverRealHTTP(t *testing.T) {
	t.Parallel()

	a, path := startedSite(t, 8124)
	base := listen(t, a)
	navigate := map[string]string{"Sec-Fetch-Site": "none", "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}

	status, _, body := browser(t, base, path, navigate)
	if status != 403 || !strings.Contains(body, "/login?next=%2Frun%2Fbakery%2F") {
		t.Fatalf("a signed-out browser must get the sign-in page, got %d %q", status, body)
	}

	for _, dest := range []string{"script", "empty", "image"} {
		forged := map[string]string{"Sec-Fetch-Site": "same-origin", "Sec-Fetch-Mode": "no-cors", "Sec-Fetch-Dest": dest}
		if got, _, _ := browser(t, base, path+"index.html", forged); got != 403 {
			t.Fatalf("a forged %s request must be forbidden, got %d", dest, got)
		}
	}

	navigate["Cookie"] = a.Cookie

	status, header, _ := browser(t, base, path, navigate)
	if status != 302 {
		t.Fatalf("the owner is sent to a ticketed address, got %d", status)
	}

	got, _, page := browser(t, base, header.Get("Location"), navigate)
	if got != 200 || !strings.Contains(page, "hello site") {
		t.Fatalf("the owner's page: %d %q", got, page)
	}
}
