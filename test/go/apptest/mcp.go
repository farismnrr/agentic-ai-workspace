package apptest

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// PublicURL is the origin configured for the in-process app.
const PublicURL = "https://example.test"

// Redirect is the OAuth redirect URI used by connector tests.
const Redirect = "https://chatgpt.com/connector_platform_oauth_redirect"

// TestPassword is a valid password shared by integration tests.
const TestPassword = "correct horse"

// SiteCommand is a local HTTP server command used by site integration tests.
const SiteCommand = `echo '<h1>hello site</h1>' > index.html && exec python3 -m http.server $PORT --bind 127.0.0.1`

// PKCEVerifier is the verifier paired with PKCEChallenge.
const PKCEVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

// PKCEChallenge is the S256 challenge for PKCEVerifier.
const PKCEChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

// FormHeaders returns headers for form-encoded requests.
func FormHeaders() map[string]string {
	return map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
}

// JSONCookie returns JSON headers scoped to a session cookie.
func JSONCookie(cookie string) map[string]string {
	return map[string]string{"Cookie": cookie, "Content-Type": "application/json"}
}

// MustNotContain fails when a response body contains any forbidden string.
func MustNotContain(t *testing.T, body []byte, unwanted ...string) {
	t.Helper()
	for _, item := range unwanted {
		if strings.Contains(string(body), item) {
			t.Errorf("response should not contain %q, got %s", item, body)
		}
	}
}

// SignedInClient registers a person and a public OAuth client.
func SignedInClient(t *testing.T, app *fiber.App) (cookie, clientID string) {
	t.Helper()
	cookie = Register(t, app, "Sari", "sari@example.com", TestPassword)
	response := Do(t, app, http.MethodPost, "/oauth/register",
		`{"client_name":"ChatGPT","redirect_uris":["`+Redirect+`"],"token_endpoint_auth_method":"none"}`,
		JSONHeaders())
	if response.Status != http.StatusCreated {
		t.Fatalf("client registration failed: %d %s", response.Status, response.Body)
	}
	clientID, _ = response.JSON(t)["client_id"].(string)
	if clientID == "" {
		t.Fatal("client registration omitted client_id")
	}
	return cookie, clientID
}

// Connect authorizes a registered client for the requested scopes.
func Connect(t *testing.T, app *fiber.App, cookie, clientID, scope string) string {
	t.Helper()
	query := url.Values{
		"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {Redirect},
		"scope": {scope}, "state": {"s1"}, "code_challenge": {PKCEChallenge},
		"code_challenge_method": {"S256"}, "resource": {PublicURL + "/api/coding"},
	}
	authorize := Do(t, app, http.MethodGet, "/oauth/authorize?"+query.Encode(), "", nil)
	location, err := url.Parse(authorize.Header.Get("Location"))
	if err != nil || authorize.Status != http.StatusFound || location.Path != "/oauth/consent" {
		t.Fatalf("authorize did not hand off to consent: %d %s", authorize.Status, authorize.Body)
	}
	headers := JSONCookie(cookie)
	decision := Do(t, app, http.MethodPost,
		"/api/oauth/requests/"+location.Query().Get("request")+"/decision", `{"approve":true}`, headers).JSON(t)
	back, err := url.Parse(stringValue(decision["redirect"]))
	if err != nil || back.Query().Get("iss") != PublicURL || back.Query().Get("state") != "s1" {
		t.Fatalf("redirect must carry issuer and state: %v", decision["redirect"])
	}
	token := Do(t, app, http.MethodPost, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")},
		"client_id": {clientID}, "redirect_uri": {Redirect}, "code_verifier": {PKCEVerifier},
		"resource": {PublicURL + "/api/coding"},
	}.Encode(), FormHeaders()).JSON(t)
	return stringValue(token["access_token"])
}

// CallTool invokes one MCP tool through the public HTTP route.
func CallTool(t *testing.T, app *fiber.App, token, name, args string) Reply {
	t.Helper()
	return Do(t, app, http.MethodPost, "/api/coding", RPC("tools/call",
		`{"name":"`+name+`","arguments":`+args+`}`), MCPHeaders(token))
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}
