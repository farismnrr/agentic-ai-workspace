package bootstrap

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestConfidentialClientDCRAndBasicAuth(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie := cookieOf(t, do(t, app, "POST", "/api/auth/register",
		`{"name":"GeminiUser","email":"gemini@example.com","password":"correct horse"}`, jsonHeaders()))
	seedWorkspace(t, app, cookie)

	// DCR without auth_method defaults to client_secret_basic and issues secret
	dcr := do(t, app, "POST", "/oauth/register",
		`{"client_name":"Google","redirect_uris":["`+redirect+`"]}`, jsonHeaders())
	if dcr.Status != 201 {
		t.Fatalf("DCR failed: %d %s", dcr.Status, dcr.Body)
	}

	payload := dcr.JSON(t)
	clientID, _ := payload["client_id"].(string)
	clientSecret, _ := payload["client_secret"].(string)
	authMethod, _ := payload["token_endpoint_auth_method"].(string)

	if clientID == "" || clientSecret == "" || authMethod != "client_secret_basic" {
		t.Fatalf("expected confidential client credentials: %v", payload)
	}

	code := authorizeAndApprove(t, app, cookie, clientID, "coding:read")

	// Token exchange using HTTP Basic auth
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret))
	headers := formHeaders()
	headers["Authorization"] = authHeader

	tokenRes := do(t, app, "POST", "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirect},
		"code_verifier": {pkceVerifier},
		"resource":      {publicURL + "/api/coding"},
	}.Encode(), headers)

	if tokenRes.Status != 200 {
		t.Fatalf("token exchange with basic auth failed: %d %s", tokenRes.Status, tokenRes.Body)
	}

	token := tokenRes.JSON(t)["access_token"].(string)
	call := rpc("tools/call", `{"name":"read_file","arguments":{"path":"hello.txt"}}`)
	mustContain(t, do(t, app, "POST", "/api/coding", call, mcpHeaders(token)).Body, "hello from the workspace")
}

func TestConfidentialClientPostBodyAuth(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie := cookieOf(t, do(t, app, "POST", "/api/auth/register",
		`{"name":"PostUser","email":"post@example.com","password":"correct horse"}`, jsonHeaders()))

	dcr := do(t, app, "POST", "/oauth/register",
		`{"client_name":"Google Post","redirect_uris":["`+redirect+`"],"token_endpoint_auth_method":"client_secret_post"}`,
		jsonHeaders())
	if dcr.Status != 201 {
		t.Fatalf("DCR failed: %d %s", dcr.Status, dcr.Body)
	}

	payload := dcr.JSON(t)
	clientID, _ := payload["client_id"].(string)
	clientSecret, _ := payload["client_secret"].(string)

	code := authorizeAndApprove(t, app, cookie, clientID, "coding:read")

	// Token exchange with wrong secret must fail
	badRes := do(t, app, "POST", "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {"wrong-secret"},
		"redirect_uri":  {redirect},
		"code_verifier": {pkceVerifier},
		"resource":      {publicURL + "/api/coding"},
	}.Encode(), formHeaders())

	if badRes.Status == 200 || !strings.Contains(string(badRes.Body), "invalid_client") {
		t.Fatalf("expected invalid_client error, got %d %s", badRes.Status, badRes.Body)
	}

	// Re-approve and exchange with correct secret in POST body
	code2 := authorizeAndApprove(t, app, cookie, clientID, "coding:read")
	tokenRes := do(t, app, "POST", "/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code2},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirect},
		"code_verifier": {pkceVerifier},
		"resource":      {publicURL + "/api/coding"},
	}.Encode(), formHeaders())

	if tokenRes.Status != 200 {
		t.Fatalf("token exchange with post secret failed: %d %s", tokenRes.Status, tokenRes.Body)
	}
}

func authorizeAndApprove(t *testing.T, app *fiber.App, cookie, clientID, scope string) string {
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
	back, _ := url.Parse(decision["redirect"].(string))

	return back.Query().Get("code")
}
