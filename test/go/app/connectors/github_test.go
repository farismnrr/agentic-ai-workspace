package connectors_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

func withGitHub(c *config.Config) {
	c.GitHub = config.GitHub{ClientID: "cid", ClientSecret: "secret", AppSlug: "my-app"}
}

func TestGitHubConnectionIsPerPersonAndStateBound(t *testing.T) {
	t.Parallel()

	app := apptest.NewEnvWith(t, withGitHub).App

	if got := apptest.Do(t, app, "GET", "/api/integrations/github", "", nil).Status; got != 401 {
		t.Fatalf("signed out people must not see it: %d", got)
	}

	sari := apptest.CookieOf(t, apptest.Do(t, app, "POST", "/api/auth/register",
		`{"name":"Sari","email":"sari@example.com","password":"`+apptest.TestPassword+`"}`, apptest.JSONHeaders()))
	bob := apptest.CookieOf(t, apptest.Do(t, app, "POST", "/api/auth/register",
		`{"name":"Bob","email":"bob@example.com","password":"`+apptest.TestPassword+`"}`, apptest.JSONHeaders()))

	status := apptest.Do(t, app, "GET", "/api/integrations/github", "", map[string]string{"Cookie": sari}).JSON(t)
	if status["configured"] != true || status["connected"] != false ||
		status["reposUrl"] != "https://github.com/apps/my-app/installations/new" {
		t.Fatalf("status: %v", status)
	}

	headers := map[string]string{"Cookie": sari, "Content-Type": "application/json"}
	start := apptest.Do(t, app, "POST", "/api/integrations/github/start", "", headers).JSON(t)
	target, _ := url.Parse(start["url"].(string)) //nolint:forcetypeassert // test

	if target.Host != "github.com" ||
		target.Query().Get("redirect_uri") != apptest.PublicURL+"/api/integrations/github/callback" ||
		target.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("authorize address: %s", target)
	}

	// Bob cannot finish the sign-in Sari started, even with her state.
	back := apptest.Do(t, app, "GET", "/api/integrations/github/callback?code=x&state="+target.Query().Get("state"),
		"", map[string]string{"Cookie": bob})
	if back.Status != 303 || !strings.HasSuffix(back.Header.Get("Location"), "?github=failed") {
		t.Fatalf("bob's callback: %d %s", back.Status, back.Header.Get("Location"))
	}
}

func TestGitHubReturnWithoutStateRestartsTheSignIn(t *testing.T) {
	t.Parallel()

	app := apptest.NewEnvWith(t, withGitHub).App
	cookie := apptest.CookieOf(t, apptest.Do(t, app, "POST", "/api/auth/register",
		`{"name":"Sari","email":"sari@example.com","password":"`+apptest.TestPassword+`"}`, apptest.JSONHeaders()))

	// What GitHub sends after the app is installed or changed there: a code, no state.
	back := apptest.Do(t, app, "GET",
		"/api/integrations/github/callback?code=from-install&installation_id=1&setup_action=install",
		"", map[string]string{"Cookie": cookie})
	target, _ := url.Parse(back.Header.Get("Location"))

	if back.Status != 303 || target.Host != "github.com" || target.Path != "/login/oauth/authorize" ||
		target.Query().Get("state") == "" || target.Query().Get("code_challenge") == "" {
		t.Fatalf("an unknown return must start our own sign-in, got %d %s", back.Status, back.Header.Get("Location"))
	}
}

func TestGitHubIsOffUntilConfigured(t *testing.T) {
	t.Parallel()

	app := apptest.NewEnv(t).App
	cookie := apptest.CookieOf(t, apptest.Do(t, app, "POST", "/api/auth/register",
		`{"name":"Sari","email":"sari@example.com","password":"`+apptest.TestPassword+`"}`, apptest.JSONHeaders()))
	headers := map[string]string{"Cookie": cookie, "Content-Type": "application/json"}

	if got := apptest.Do(t, app, "POST", "/api/integrations/github/start", "", headers).Status; got != 409 {
		t.Fatalf("start without a GitHub App: %d", got)
	}
}

func TestImportingARepoNeedsTheOwnerAGoodNameAndAConnection(t *testing.T) {
	t.Parallel()

	a := apptest.RigOn(t, apptest.NewEnvWith(t, withGitHub))
	path := "/api/projects/" + a.Project + "/import"

	for _, bad := range []string{`{"repo":""}`, `{"repo":"no-slash"}`, `{"repo":"a/b; rm -rf /"}`, `{"repo":"../x/y"}`} {
		if got := apptest.Do(t, a.Environment.App, "POST", path, bad, apptest.JSONCookie(a.Cookie)).Status; got != 400 {
			t.Errorf("%s must be refused as a bad name, got %d", bad, got)
		}
	}

	if got := apptest.Do(t, a.Environment.App, "POST", path, `{"repo":"ann/shop"}`,
		apptest.JSONCookie(a.Cookie)); got.Status != 409 {
		t.Errorf("without a GitHub connection the import is refused: %d %s", got.Status, got.Body)
	}

	other := apptest.CookieOf(t, apptest.Do(t, a.Environment.App, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"`+apptest.TestPassword+`"}`, apptest.JSONHeaders()))
	if got := apptest.Do(t, a.Environment.App, "POST", path, `{"repo":"ann/shop"}`,
		apptest.JSONCookie(other)).Status; got != 404 {
		t.Errorf("someone else's project is not found: %d", got)
	}

	if got := apptest.Do(t, a.Environment.App, "GET", "/api/integrations/github/repos", "",
		apptest.JSONCookie(a.Cookie)).Status; got != 409 {
		t.Errorf("the repo list needs a connection too: %d", got)
	}
}
