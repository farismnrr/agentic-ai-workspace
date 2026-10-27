package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"

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

	cookie = cookieOf(t, do(t, app, "POST", "/api/auth/register",
		`{"name":"Sari","email":"sari@example.com","password":"correct horse"}`, jsonHeaders()))

	dcr := do(t, app, "POST", "/oauth/register",
		`{"client_name":"ChatGPT","redirect_uris":["`+redirect+`"],"token_endpoint_auth_method":"none"}`, jsonHeaders())
	if dcr.Status != 201 {
		t.Fatalf("DCR failed: %d %s", dcr.Status, dcr.Body)
	}

	id, _ := dcr.JSON(t)["client_id"].(string)

	seedWorkspace(t, app, cookie, "ChatGPT")

	return cookie, id
}

func TestUnauthenticatedCallPointsToResourceMetadata(t *testing.T) {
	t.Parallel()

	res := do(t, newApp(t), "POST", "/api/coding", rpc("tools/list", "{}"), mcpHeaders(""))
	want := `resource_metadata="https://example.test/.well-known/oauth-protected-resource/api/coding"`

	if res.Status != 401 || !strings.Contains(res.Header.Get("WWW-Authenticate"), want) {
		t.Fatalf("expected 401 with resource_metadata, got %d %v", res.Status, res.Header)
	}
}

func TestProtectedResourceMetadata(t *testing.T) {
	t.Parallel()

	body := do(t, newApp(t), "GET", "/.well-known/oauth-protected-resource/api/coding", "", nil).Body

	mustContain(t, body, `"resource":"`+publicURL+`/api/coding"`, `"authorization_servers":["`+publicURL+`"]`,
		`"scopes_supported":["coding:read","coding:approve","coding:write","coding:exec"]`,
		`"bearer_methods_supported":["header"]`)
}

func TestDiscoveryDocumentsMatch(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	oauth := do(t, app, "GET", "/.well-known/oauth-authorization-server", "", nil)
	oidc := do(t, app, "GET", "/.well-known/openid-configuration", "", nil)

	if string(oauth.Body) != string(oidc.Body) {
		t.Fatal("RFC 8414 and OIDC discovery documents must be identical")
	}
}

func TestAuthorizationServerAdvertisesRegistrationAndPKCE(t *testing.T) {
	t.Parallel()

	body := do(t, newApp(t), "GET", "/.well-known/oauth-authorization-server", "", nil).Body

	mustContain(t, body, `"issuer":"`+publicURL+`"`, `"client_id_metadata_document_supported":true`,
		`"authorization_response_iss_parameter_supported":true`, `"registration_endpoint":"`+publicURL+`/oauth/register"`,
		`"authorization_endpoint":"`+publicURL+`/oauth/authorize"`, `"jwks_uri":"`+publicURL+`/oauth/jwks.json"`,
		`"code_challenge_methods_supported":["S256"]`,
		`"token_endpoint_auth_methods_supported":["none","client_secret_basic","client_secret_post"]`, "coding:read")
}

func TestJWKSPublishesOneKey(t *testing.T) {
	t.Parallel()

	var keys struct {
		Keys []map[string]string `json:"keys"`
	}

	body := do(t, newApp(t), "GET", "/oauth/jwks.json", "", nil).Body
	if err := json.Unmarshal(body, &keys); err != nil || len(keys.Keys) != 1 || keys.Keys[0]["alg"] != "ES256" {
		t.Fatalf("JWKS must publish the ES256 signing key: %s", body)
	}
}

func TestToolsAreListedWithSecuritySchemes(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	token := connect(t, app, cookie, clientID, "openid email coding:read")

	tools := do(t, app, "POST", "/api/coding", rpc("tools/list", "{}"), mcpHeaders(token))
	mustContain(t, tools.Body, "list_files", "read_file", "search_code", "whoami", "securitySchemes")
}

func TestReadFileThroughMCP(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	token := connect(t, app, cookie, clientID, "coding:read")

	call := rpc("tools/call", `{"name":"read_file","arguments":{"path":"hello.txt"}}`)
	mustContain(t, do(t, app, "POST", "/api/coding", call, mcpHeaders(token)).Body, "hello from the workspace")

	secret := rpc("tools/call", `{"name":"read_file","arguments":{"path":".env"}}`)
	mustContain(t, do(t, app, "POST", "/api/coding", secret, mcpHeaders(token)).Body, "not available")
}

func TestWhoamiReturnsAccountID(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	token := connect(t, app, cookie, clientID, "coding:read")

	me, _ := do(t, app, "GET", "/api/auth/me", "", map[string]string{"Cookie": cookie}).JSON(t)["id"].(string)
	who := do(t, app, "POST", "/api/coding", rpc("tools/call", `{"name":"whoami","arguments":{}}`), mcpHeaders(token))

	mustContain(t, who.Body, me)
}

func TestMissingScopeGetsInsufficientScope(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	narrow := connect(t, app, cookie, clientID, "openid")

	denied := do(t, app, "POST", "/api/coding", rpc("tools/list", "{}"), mcpHeaders(narrow))
	if denied.Status != 403 || !strings.Contains(denied.Header.Get("WWW-Authenticate"), `error="insufficient_scope"`) {
		t.Fatalf("expected 403 insufficient_scope, got %d %v", denied.Status, denied.Header)
	}
}

func TestRevokedGrantStopsWorkingAtOnce(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	token := connect(t, app, cookie, clientID, "coding:read")
	headers := map[string]string{"Cookie": cookie}

	var grants []map[string]any
	listed := do(t, app, "GET", "/api/oauth/grants", "", headers)
	if err := json.Unmarshal(listed.Body, &grants); err != nil || len(grants) != 1 {
		t.Fatalf("the connection should be listed: %v %v", grants, err)
	}

	id, _ := grants[0]["id"].(string)
	if removed := do(t, app, "DELETE", "/api/oauth/grants/"+id, "", headers); removed.Status != 204 {
		t.Fatalf("revoke failed: %d", removed.Status)
	}

	if after := do(t, app, "POST", "/api/coding", rpc("tools/list", "{}"), mcpHeaders(token)); after.Status != 401 {
		t.Fatalf("revoked grants must get 401, got %d", after.Status)
	}
}

func TestUnknownAPIPathIs404JSONNotSPA(t *testing.T) {
	t.Parallel()

	res := do(t, newApp(t), "GET", "/api/nope", "", nil)
	if res.Status != 404 || !strings.Contains(string(res.Body), "not_found") {
		t.Fatalf("got %d %s", res.Status, res.Body)
	}
}
