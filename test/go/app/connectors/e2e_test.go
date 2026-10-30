package connectors_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"

	"github.com/gofiber/fiber/v3"
)

func rpc(method, params string) string {
	return `{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`
}

func mcpHeaders(token string) map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json", "Accept": "application/json, text/event-stream",
		"MCP-Protocol-Version": "2025-11-25",
	}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}

	return headers
}

func mustContain(t *testing.T, body []byte, wants ...string) {
	t.Helper()

	for _, want := range wants {
		if !strings.Contains(string(body), want) {
			t.Errorf("response should contain %q, got %s", want, body)
		}
	}
}

func signedInClient(t *testing.T, app *fiber.App) (cookie, clientID string) {
	t.Helper()

	cookie = apptest.CookieOf(t, apptest.Do(t, app, "POST", "/api/auth/register",
		`{"name":"Sari","email":"sari@example.com","password":"correct horse"}`, apptest.JSONHeaders()))

	dcr := apptest.Do(t, app, "POST", "/oauth/register",
		`{"client_name":"ChatGPT","redirect_uris":["`+apptest.Redirect+`"],"token_endpoint_auth_method":"none"}`,
		apptest.JSONHeaders())
	if dcr.Status != 201 {
		t.Fatalf("DCR failed: %d %s", dcr.Status, dcr.Body)
	}

	id, _ := dcr.JSON(t)["client_id"].(string)

	return cookie, id
}

func TestUnauthenticatedCallPointsToResourceMetadata(t *testing.T) {
	t.Parallel()

	res := apptest.Do(t, apptest.NewEnv(t).App, "POST", "/api/coding", rpc("tools/list", "{}"), mcpHeaders(""))
	want := `resource_metadata="https://example.test/.well-known/oauth-protected-resource/api/coding"`

	if res.Status != 401 || !strings.Contains(res.Header.Get("WWW-Authenticate"), want) {
		t.Fatalf("expected 401 with resource_metadata, got %d %v", res.Status, res.Header)
	}
}

func TestProtectedResourceMetadata(t *testing.T) {
	t.Parallel()

	body := apptest.Do(t, apptest.NewEnv(t).App, "GET", "/.well-known/oauth-protected-resource/api/coding", "", nil).Body

	mustContain(t, body, `"resource":"`+apptest.PublicURL+`/api/coding"`,
		`"authorization_servers":["`+apptest.PublicURL+`"]`,
		`"scopes_supported":["coding:read","coding:approve","coding:write","coding:exec"]`,
		`"bearer_methods_supported":["header"]`)
}

func TestDiscoveryDocumentsMatch(t *testing.T) {
	t.Parallel()

	app := apptest.NewEnv(t).App
	oauth := apptest.Do(t, app, "GET", "/.well-known/oauth-authorization-server", "", nil)
	oidc := apptest.Do(t, app, "GET", "/.well-known/openid-configuration", "", nil)

	if string(oauth.Body) != string(oidc.Body) {
		t.Fatal("RFC 8414 and OIDC discovery documents must be identical")
	}
}

func TestAuthorizationServerAdvertisesRegistrationAndPKCE(t *testing.T) {
	t.Parallel()

	body := apptest.Do(t, apptest.NewEnv(t).App, "GET", "/.well-known/oauth-authorization-server", "", nil).Body

	mustContain(t, body, `"issuer":"`+apptest.PublicURL+`"`, `"client_id_metadata_document_supported":true`,
		`"authorization_response_iss_parameter_supported":true`,
		`"registration_endpoint":"`+apptest.PublicURL+`/oauth/register"`,
		`"authorization_endpoint":"`+apptest.PublicURL+`/oauth/authorize"`,
		`"jwks_uri":"`+apptest.PublicURL+`/oauth/jwks.json"`,
		`"code_challenge_methods_supported":["S256"]`,
		`"token_endpoint_auth_methods_supported":["none","client_secret_basic","client_secret_post"]`, "coding:read")
}

func TestJWKSPublishesOneKey(t *testing.T) {
	t.Parallel()

	var keys struct {
		Keys []map[string]string `json:"keys"`
	}

	body := apptest.Do(t, apptest.NewEnv(t).App, "GET", "/oauth/jwks.json", "", nil).Body
	if err := json.Unmarshal(body, &keys); err != nil || len(keys.Keys) != 1 || keys.Keys[0]["alg"] != "ES256" {
		t.Fatalf("JWKS must publish the ES256 signing key: %s", body)
	}
}

func TestToolsAreListedWithSecuritySchemes(t *testing.T) {
	t.Parallel()

	app := apptest.NewEnv(t).App
	cookie, clientID := signedInClient(t, app)
	token := apptest.Connect(t, app, cookie, clientID, "openid email coding:read")

	tools := apptest.Do(t, app, "POST", "/api/coding", rpc("tools/list", "{}"), mcpHeaders(token))
	mustContain(t, tools.Body, "list_files", "read_file", "search_code", "whoami", "securitySchemes")
}

func TestReadFileThroughMCP(t *testing.T) {
	t.Parallel()
	env := apptest.NewEnv(t)
	app := env.App
	cookie, clientID := signedInClient(t, app)
	apptest.SeedWorkspace(t, env, cookie, "ChatGPT")
	token := apptest.Connect(t, app, cookie, clientID, "coding:read")

	// No session means no role attribution: fail closed before reading any file.
	unaffiliated := rpc("tools/call", `{"name":"read_file","arguments":{"path":"hello.txt"}}`)
	mustContain(t, apptest.Do(t, app, "POST", "/api/coding", unaffiliated, mcpHeaders(token)).Body,
		`"isError":true`, "Start with begin_work")

	begin := apptest.CallTool(t, app, token, "begin_work", `{"role":"researcher","summary":"Read the workspace for research"}`)
	var envelope struct {
		Result struct {
			Structured struct {
				WorkID string `json:"work_id"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(begin.Body, &envelope); err != nil || envelope.Result.Structured.WorkID == "" {
		t.Fatalf("begin_work did not return a tracked session: %v %s", err, begin.Body)
	}
	workID := envelope.Result.Structured.WorkID
	call := rpc("tools/call", `{"name":"read_file","arguments":{"work_id":"`+workID+`","path":"hello.txt"}}`)
	mustContain(t, apptest.Do(t, app, "POST", "/api/coding", call, mcpHeaders(token)).Body, "hello from the workspace")
	secret := rpc("tools/call", `{"name":"read_file","arguments":{"work_id":"`+workID+`","path":".env"}}`)
	mustContain(t, apptest.Do(t, app, "POST", "/api/coding", secret, mcpHeaders(token)).Body, "not available")
}

func TestWhoamiReturnsAccountID(t *testing.T) {
	t.Parallel()

	app := apptest.NewEnv(t).App
	cookie, clientID := signedInClient(t, app)
	token := apptest.Connect(t, app, cookie, clientID, "coding:read")

	me, _ := apptest.Do(t, app, "GET", "/api/auth/me", "", map[string]string{"Cookie": cookie}).JSON(t)["id"].(string)
	who := apptest.Do(t, app, "POST", "/api/coding", rpc("tools/call", `{"name":"whoami","arguments":{}}`),
		mcpHeaders(token))

	mustContain(t, who.Body, me)
}

func TestUnknownAPIPathIs404JSONNotSPA(t *testing.T) {
	t.Parallel()

	res := apptest.Do(t, apptest.NewEnv(t).App, "GET", "/api/nope", "", nil)
	if res.Status != 404 || !strings.Contains(string(res.Body), "not_found") {
		t.Fatalf("got %d %s", res.Status, res.Body)
	}
}
