package bootstrap

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

func TestSpacesListNeedsSessionAndDescribesTools(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	if anon := do(t, app, "GET", "/api/spaces", "", nil); anon.Status != 401 {
		t.Fatalf("spaces are private to signed-in people, got %d", anon.Status)
	}

	cookie, _ := signedInClient(t, app)
	spaces := do(t, app, "GET", "/api/spaces", "", map[string]string{"Cookie": cookie})

	mustContain(t, spaces.Body, `"url":"`+publicURL+`/api/coding"`, `"name":"read_file"`, `"readOnly":true`,
		`"id":"coding:read"`, `"id":"coding:approve"`, "Read files and search code")
}

func TestTheConnectorAddressFollowsConnectorURLNotTheAppAddress(t *testing.T) {
	t.Parallel()

	const connector = "https://connect.example.test"

	env := newEnvWith(t, func(cfg *config.Config) { cfg.ConnectorURL = connector })
	cookie, _ := signedInClient(t, env.app)

	spaces := do(t, env.app, "GET", "/api/spaces", "", map[string]string{"Cookie": cookie})
	mustContain(t, spaces.Body, `"url":"`+connector+`/api/coding"`)
	mustNotContain(t, spaces.Body, publicURL)

	resource := do(t, env.app, "GET", "/.well-known/oauth-protected-resource/api/coding", "", nil)
	mustContain(t, resource.Body, `"resource":"`+connector+`/api/coding"`, `"authorization_servers":["`+connector+`"]`)

	issuer := do(t, env.app, "GET", "/.well-known/oauth-authorization-server", "", nil)
	mustContain(t, issuer.Body, `"issuer":"`+connector+`"`, connector+"/oauth/token")

	challenge := do(t, env.app, "POST", "/api/coding", rpc("tools/list", "{}"), mcpHeaders(""))
	mustContain(t, []byte(challenge.Header.Get("WWW-Authenticate")),
		`resource_metadata="`+connector+`/.well-known/oauth-protected-resource/api/coding"`)
}
