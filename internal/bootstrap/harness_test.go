package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// TestMain lets the test binary act as the sandbox init, as the server binary does.
func TestMain(m *testing.M) {
	sandbox.RunIfChild()
	os.Exit(m.Run())
}

const (
	publicURL     = "https://example.test"
	pkceVerifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	pkceChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	redirect      = "https://chatgpt.com/connector_platform_oauth_redirect"
)

type reply struct {
	Header http.Header
	Body   []byte
	Status int
}

func (r reply) JSON(t *testing.T) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal(r.Body, &out); err != nil {
		t.Fatalf("not JSON (%d): %s", r.Status, r.Body)
	}

	return out
}

func newApp(t *testing.T) *fiber.App {
	t.Helper()

	app, _ := newWorkspace(t)

	return app
}

// newWorkspace starts the app on a temporary workspace and returns its folder too.
func newWorkspace(t *testing.T) (*fiber.App, string) {
	t.Helper()

	env := newEnv(t)

	return env.app, env.root
}

// testEnv is a running app with the folders it works in.
type testEnv struct {
	app     *fiber.App
	root    string
	dataDir string
}

func newEnv(t *testing.T) testEnv {
	t.Helper()

	return newEnvWith(t, nil)
}

// newEnvWith is newEnv with a chance to adjust the configuration first.
func newEnvWith(t *testing.T, adjust func(*config.Config)) testEnv {
	t.Helper()

	root := filepath.Join(t.TempDir(), "workspaces")

	cfg := config.Config{
		Port: "0", PublicURL: publicURL, DataDir: t.TempDir(), WorkspacesDir: root, CookieSecure: true,
		SandboxDir: testSandboxDir(), SandboxRelaxed: true, MaxSandboxes: 4,
		AllowedOrigins: []string{"http://localhost:8081", "http://127.0.0.1:8081"},
		CORSOrigins:    []string{"http://localhost:8081"},
	}

	if adjust != nil {
		adjust(&cfg)
	}

	db, err := database.Open(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })

	mounts, stop, err := wire(db, cfg)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(stop)

	app := httpserver.New(fstest.MapFS{"index.html": {Data: []byte("<html>")}}, httpserver.Settings{}, mounts...)

	workspaceRoots.Store(app, root)

	return testEnv{app: app, root: root, dataDir: cfg.DataDir}
}

// connect runs the whole OAuth dance for scope and returns the access token.
func connect(t *testing.T, app *fiber.App, cookie, clientID, scope string) string {
	t.Helper()

	query := url.Values{
		"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {redirect}, "scope": {scope},
		"state": {"s1"}, "code_challenge": {pkceChallenge}, "code_challenge_method": {"S256"},
		"resource": {publicURL + "/api/coding"},
	}

	authorize := do(t, app, "GET", "/oauth/authorize?"+query.Encode(), "", nil)
	location, _ := url.Parse(authorize.Header.Get("Location"))

	if authorize.Status != http.StatusFound || location.Path != "/oauth/consent" {
		t.Fatalf("authorize should hand off to consent: %d %s", authorize.Status, authorize.Body)
	}

	headers := jsonHeaders()
	headers["Cookie"] = cookie

	decision := do(t, app, "POST", "/api/oauth/requests/"+location.Query().Get("request")+"/decision",
		`{"approve":true}`, headers).JSON(t)
	back, _ := url.Parse(decision["redirect"].(string)) //nolint:forcetypeassert // test

	if back.Query().Get("iss") != publicURL || back.Query().Get("state") != "s1" {
		t.Fatalf("redirect must carry iss and state: %v", decision["redirect"])
	}

	token := do(t, app, "POST", "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")}, "client_id": {clientID},
		"redirect_uri": {redirect}, "code_verifier": {pkceVerifier}, "resource": {publicURL + "/api/coding"},
	}.Encode(), formHeaders()).JSON(t)

	return token["access_token"].(string) //nolint:forcetypeassert // test
}

// fileText reads a file of the test workspace.
func fileText(t *testing.T, root, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, name)) //nolint:gosec // reads the test's own temp workspace
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}
