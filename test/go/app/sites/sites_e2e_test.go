package sites_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

const siteCommand = "python3 -m http.server 8123 --bind 127.0.0.1"

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
		apptest.ToolCall("html", "write_file", map[string]any{"path": "index.html", "content": "<h1>hello site</h1>"}),
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

	// Page navigations to missing sites use the SPA not-found screen, while
	// subresource requests receive a plain 404. Neither may serve old content.
	stopped := apptest.Do(t, a.Environment.App, "GET", url, "", page)
	if stopped.Status != 302 || stopped.Header.Get("Location") != "/not-found" {
		t.Fatalf("a stopped site must redirect to not-found, got %d %q",
			stopped.Status, stopped.Header.Get("Location"))
	}
	if got := apptest.Do(t, a.Environment.App, "GET", url, "", script).Status; got != 404 {
		t.Fatalf("a stopped site's resources must be gone, got %d", got)
	}
}

func TestSiteRuntimeSQLiteDirectoriesPersistAndWorkspaceStaysReadOnly(t *testing.T) {
	t.Parallel()
	image := os.Getenv("SANDBOX_TEST_IMAGE")
	if image == "" {
		t.Skip("set SANDBOX_TEST_IMAGE to the toolbox image to run this test")
	}

	a := apptest.NewAgentRigWith(t, func(c *config.Config) { c.SandboxImage = image },
		apptest.ToolCall("s1", "start_site", map[string]any{"command": "node server.js", "port": 8127}),
		apptest.FinishCall("The site is running"),
	)
	a.PutFile(t, "data/preview-runs/keep.txt", "before\n")
	a.PutFile(t, "server.js", sqliteSiteFixture)
	id := a.Start(t, "Run my app")
	a.WaitApproving(t, id, "done")

	url := "/run/bakery/"
	assertSiteRuntimeState(t, a, url, 1)
	apptest.Do(t, a.Environment.App, "POST", "/api/sites/"+a.Project+"/stop", "",
		apptest.JSONCookie(a.Cookie))
	restarted := apptest.Do(t, a.Environment.App, "POST", "/api/sites/"+a.Project+"/start", "",
		apptest.JSONCookie(a.Cookie))
	if restarted.Status != 200 {
		t.Fatalf("site did not restart: %d %s", restarted.Status, restarted.Body)
	}
	assertSiteRuntimeState(t, a, url, 2)
}

const sqliteSiteFixture = `const { spawn, execFileSync } = require('node:child_process')
const fs = require('node:fs')
const http = require('node:http')
const sqlite = "import sqlite3,time,pathlib\np='/workspace/data/preview-runs/site.sqlite'\nc=sqlite3.connect(p)\nc.execute('pragma journal_mode=wal')\nc.execute('create table if not exists runs (id integer primary key)')\nc.execute('insert into runs default values')\nc.commit()\npathlib.Path('/workspace/.atja-runtime/starts.txt').open('a').write('started\\n')\nprint('ready',flush=True)\ntime.sleep(600)"
const database = spawn('python3', ['-c', sqlite], { stdio: ['ignore', 'pipe', 'inherit'] })
database.stdout.once('data', () => {
  fs.appendFileSync('/workspace/data/preview-runs/keep.txt', 'updated\n')
  try { fs.writeFileSync('/workspace/read-only-probe', 'must fail') } catch {}
  http.createServer((_request, response) => {
    const count = Number(execFileSync('python3', ['-c', "import sqlite3;print(sqlite3.connect('/workspace/data/preview-runs/site.sqlite').execute('select count(*) from runs').fetchone()[0])"]).toString().trim())
    const state = {
      db: fs.existsSync('/workspace/data/preview-runs/site.sqlite'),
      wal: fs.existsSync('/workspace/data/preview-runs/site.sqlite-wal'),
      shm: fs.existsSync('/workspace/data/preview-runs/site.sqlite-shm'),
      starts: fs.readFileSync('/workspace/.atja-runtime/starts.txt', 'utf8').trim().split('\n').length,
      existing: fs.readFileSync('/workspace/data/preview-runs/keep.txt', 'utf8') === 'before\nupdated\n',
      count,
      denied: fs.existsSync('/workspace/read-only-probe')
    }
    response.end(JSON.stringify(state))
  }).listen(8127, '127.0.0.1')
})`

func assertSiteRuntimeState(t *testing.T, a apptest.AgentRig, url string, wantCount int) {
	t.Helper()
	headers := map[string]string{"Cookie": a.Cookie, "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}
	redirect := apptest.Do(t, a.Environment.App, "GET", url, "", headers)
	if redirect.Status != 302 {
		t.Fatalf("site did not issue its private address: %d %s", redirect.Status, redirect.Body)
	}
	page := apptest.Do(t, a.Environment.App, "GET", redirect.Header.Get("Location"), "", headers)
	if page.Status != 200 {
		t.Fatalf("site did not answer: %d %s", page.Status, page.Body)
	}
	var state struct {
		DB       bool `json:"db"`
		WAL      bool `json:"wal"`
		SHM      bool `json:"shm"`
		Starts   int  `json:"starts"`
		Existing bool `json:"existing"`
		Count    int  `json:"count"`
		Denied   bool `json:"denied"`
	}
	if err := json.Unmarshal([]byte(page.Body), &state); err != nil {
		t.Fatalf("site response is not runtime state: %v %q", err, page.Body)
	}
	if !state.DB || !state.WAL || !state.SHM || state.Starts != wantCount || !state.Existing ||
		state.Count != wantCount || state.Denied {
		t.Fatalf("unexpected site runtime state after %d start(s): %+v", wantCount, state)
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
