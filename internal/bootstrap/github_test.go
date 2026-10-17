package bootstrap

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/adapter/githubapi"
	githubsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/adapter/sqlite"
	githubapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/application"
	githubdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
)

func withGitHub(c *config.Config) {
	c.GitHub = config.GitHub{ClientID: "cid", ClientSecret: "secret", AppSlug: "my-app"}
}

func TestGitHubConnectionIsPerPersonAndStateBound(t *testing.T) {
	t.Parallel()

	app := newEnvWith(t, withGitHub).app

	if got := do(t, app, "GET", "/api/integrations/github", "", nil).Status; got != 401 {
		t.Fatalf("signed out people must not see it: %d", got)
	}

	sari := cookieOf(t, do(t, app, "POST", "/api/auth/register",
		`{"name":"Sari","email":"sari@example.com","password":"`+testPassword+`"}`, jsonHeaders()))
	bob := cookieOf(t, do(t, app, "POST", "/api/auth/register",
		`{"name":"Bob","email":"bob@example.com","password":"`+testPassword+`"}`, jsonHeaders()))

	status := do(t, app, "GET", "/api/integrations/github", "", map[string]string{"Cookie": sari}).JSON(t)
	if status["configured"] != true || status["connected"] != false ||
		status["reposUrl"] != "https://github.com/apps/my-app/installations/new" {
		t.Fatalf("status: %v", status)
	}

	headers := map[string]string{"Cookie": sari, "Content-Type": "application/json"}
	start := do(t, app, "POST", "/api/integrations/github/start", "", headers).JSON(t)
	target, _ := url.Parse(start["url"].(string)) //nolint:forcetypeassert // test

	if target.Host != "github.com" || target.Query().Get("redirect_uri") != publicURL+githubCallback ||
		target.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("authorize address: %s", target)
	}

	// Bob cannot finish the sign-in Sari started, even with her state.
	back := do(t, app, "GET", "/api/integrations/github/callback?code=x&state="+target.Query().Get("state"),
		"", map[string]string{"Cookie": bob})
	if back.Status != 303 || !strings.HasSuffix(back.Header.Get("Location"), "?github=failed") {
		t.Fatalf("bob's callback: %d %s", back.Status, back.Header.Get("Location"))
	}
}

func TestGitHubReturnWithoutStateRestartsTheSignIn(t *testing.T) {
	t.Parallel()

	app := newEnvWith(t, withGitHub).app
	cookie := cookieOf(t, do(t, app, "POST", "/api/auth/register",
		`{"name":"Sari","email":"sari@example.com","password":"`+testPassword+`"}`, jsonHeaders()))

	// What GitHub sends after the app is installed or changed there: a code, no state.
	back := do(t, app, "GET", "/api/integrations/github/callback?code=from-install&installation_id=1&setup_action=install",
		"", map[string]string{"Cookie": cookie})
	target, _ := url.Parse(back.Header.Get("Location"))

	if back.Status != 303 || target.Host != "github.com" || target.Path != "/login/oauth/authorize" ||
		target.Query().Get("state") == "" || target.Query().Get("code_challenge") == "" {
		t.Fatalf("an unknown return must start our own sign-in, got %d %s", back.Status, back.Header.Get("Location"))
	}
}

func TestGitHubIsOffUntilConfigured(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie := cookieOf(t, do(t, app, "POST", "/api/auth/register",
		`{"name":"Sari","email":"sari@example.com","password":"`+testPassword+`"}`, jsonHeaders()))
	headers := map[string]string{"Cookie": cookie, "Content-Type": "application/json"}

	if got := do(t, app, "POST", "/api/integrations/github/start", "", headers).Status; got != 409 {
		t.Fatalf("start without a GitHub App: %d", got)
	}
}

type fixedOwners map[string]string

func (f fixedOwners) Owner(_ context.Context, id string) (string, string, error) {
	if owner, ok := f[id]; ok {
		return owner, "project", nil
	}

	return "", "", errNoProject
}

var errNoProject = errors.New("no such project")

func TestCommandsActOnlyAsTheProjectOwner(t *testing.T) {
	t.Parallel()

	env := newEnvWith(t, withGitHub)
	register := func(name string) string {
		return cookieOf(t, do(t, env.app, "POST", "/api/auth/register",
			`{"name":"`+name+`","email":"`+name+`@example.com","password":"`+testPassword+`"}`, jsonHeaders()))
	}
	sari, bob := register("sari"), register("bob")
	idOf := func(cookie string) string {
		id, _ := do(t, env.app, "GET", "/api/auth/me", "", map[string]string{"Cookie": cookie}).JSON(t)["id"].(string)

		return id
	}

	value, _ := testStores.Load(env.app)
	store, _ := value.(testStore)

	sealer, err := secretbox.Open(store.dataDir)
	if err != nil {
		t.Fatal(err)
	}

	sealed, _ := sealer.Encrypt([]byte("ghu_sari"))
	record := githubapp.Record{
		UserID: idOf(sari), Profile: githubdomain.Profile{ID: 1, Login: "sari"}, AccessSealed: sealed,
		RefreshSealed: sealed, AccessExpires: time.Now().Add(time.Hour),
	}

	gh := githubsqlite.NewStore(store.db)
	if err = gh.Save(context.Background(), record, time.Now()); err != nil {
		t.Fatal(err)
	}

	client := githubapi.New("cid", "s", "", time.Now)
	service := githubapp.NewService(gh, client, sealer, opaque.Generator{}, time.Now, true)
	tools := &agentTools{}
	tools.useGitHub(service, fixedOwners{"sari-project": idOf(sari), "bob-project": idOf(bob)})

	got, ok := tools.git("sari-project")(context.Background())
	if !ok || got.Token != "ghu_sari" || got.Email != "1+sari@users.noreply.github.com" {
		t.Fatalf("the owner's own project: %+v %v", got, ok)
	}

	for _, project := range []string{"bob-project", "unknown-project"} {
		if leaked, found := tools.git(project)(context.Background()); found || leaked.Token != "" {
			t.Fatalf("%s must not get anyone's token: %+v", project, leaked)
		}
	}
}

func TestImportingARepoNeedsTheOwnerAGoodNameAndAConnection(t *testing.T) {
	t.Parallel()

	a := rigOn(t, newEnvWith(t, withGitHub))
	path := "/api/projects/" + a.project + "/import"

	for _, bad := range []string{`{"repo":""}`, `{"repo":"no-slash"}`, `{"repo":"a/b; rm -rf /"}`, `{"repo":"../x/y"}`} {
		if got := do(t, a.env.app, "POST", path, bad, jsonCookie(a.cookie)).Status; got != 400 {
			t.Errorf("%s must be refused as a bad name, got %d", bad, got)
		}
	}

	if got := do(t, a.env.app, "POST", path, `{"repo":"ann/shop"}`, jsonCookie(a.cookie)); got.Status != 409 {
		t.Errorf("without a GitHub connection the import is refused: %d %s", got.Status, got.Body)
	}

	other := cookieOf(t, do(t, a.env.app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"`+testPassword+`"}`, jsonHeaders()))
	if got := do(t, a.env.app, "POST", path, `{"repo":"ann/shop"}`, jsonCookie(other)).Status; got != 404 {
		t.Errorf("someone else's project is not found: %d", got)
	}

	if got := do(t, a.env.app, "GET", "/api/integrations/github/repos", "", jsonCookie(a.cookie)).Status; got != 409 {
		t.Errorf("the repo list needs a connection too: %d", got)
	}
}
