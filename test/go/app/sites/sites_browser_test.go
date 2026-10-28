package sites_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/webassets"
	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

// probePage is what the site serves in the browser test: it tries everything a hostile page would
// try against the app around it and writes what happened into the page.
const probePage = `<h1>hello site</h1><pre id="out">running</pre><script>
(async () => {
  const r = [];
  try { r.push('cookie=' + JSON.stringify(document.cookie)); } catch (e) { r.push('cookie=blocked:' + e.name); }
  try { localStorage.setItem('x', '1'); r.push('storage=allowed'); } catch (e) { r.push('storage=blocked:' + e.name); }
  try { const res = await fetch('/api/projects', { credentials: 'include' }); r.push('api=' + res.status); }
  catch (e) { r.push('api=blocked:' + e.name); }
  try { await navigator.serviceWorker.register('sw.js'); r.push('sw=allowed'); }
  catch (e) { r.push('sw=blocked:' + e.name); }
  document.getElementById('out').textContent = r.join(' | ');
})();
</script>`

// TestServeSiteForABrowser starts the whole app with the real frontend and a running private
// site, then waits for an external browser run (Playwright) to finish. It does nothing unless
// SITES_BROWSER_DIR is set. It writes info.json there and stops when a file named done appears.
func TestServeSiteForABrowser(t *testing.T) {
	dir, image := os.Getenv("SITES_BROWSER_DIR"), os.Getenv("SANDBOX_TEST_IMAGE")
	if dir == "" || image == "" {
		t.Skip("set SITES_BROWSER_DIR and SANDBOX_TEST_IMAGE to serve a site for a browser run")
	}

	frontend, err := webassets.Assets()
	if err != nil {
		t.Fatal(err)
	}

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	base := "http://" + ln.Addr().String()
	env := apptest.NewEnvServing(t, func(c *config.Config) {
		c.SandboxImage, c.PublicURL, c.CookieSecure = image, base, false
		c.AllowedOrigins, c.CORSOrigins = []string{base}, []string{base}
	}, frontend)

	command := "echo " + base64.StdEncoding.EncodeToString([]byte(probePage)) +
		" | base64 -d > index.html && exec python3 -m http.server $PORT --bind 127.0.0.1"
	a := apptest.RigOn(t, env,
		apptest.ToolCall("s1", "start_site", map[string]any{"command": command, "port": 8125}),
		apptest.FinishCall("Your page is running"),
	)

	go func() { _ = env.App.Listener(ln) }()

	t.Cleanup(func() { _ = env.App.Shutdown() })

	id := a.Start(t, "Show me a page")
	a.WaitApproving(t, id, "done")

	info, _ := json.Marshal(map[string]string{
		"base": base, "path": siteLink(t, a, id), "email": "sari@example.com", "password": apptest.TestPassword,
	})
	if err = os.WriteFile(filepath.Join(dir, "info.json"), info, 0o600); err != nil { //nolint:gosec // test-only folder
		t.Fatal(err)
	}

	for deadline := time.Now().Add(5 * time.Minute); time.Now().Before(deadline); time.Sleep(500 * time.Millisecond) {
		if _, statErr := os.Stat(filepath.Join(dir, "done")); statErr == nil { //nolint:gosec // test-only folder
			return
		}
	}
}

func siteLink(t *testing.T, a apptest.AgentRig, taskID string) string {
	t.Helper()

	for _, event := range a.Thread(t, taskID) {
		if event.Kind == "site" {
			return event.URL
		}
	}

	t.Fatal("the thread has no site link")

	return ""
}
