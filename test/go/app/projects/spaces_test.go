package projects_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

func TestSpacesListNeedsSessionAndDescribesTools(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	app := env.App

	if anon := apptest.Do(t, app, "GET", "/api/spaces", "", nil); anon.Status != 401 {
		t.Fatalf("spaces are private to signed-in people, got %d", anon.Status)
	}

	cookie := apptest.Register(t, app, "Sari", "sari@example.com", "correct horse")
	spaces := apptest.Do(t, app, "GET", "/api/spaces", "", map[string]string{"Cookie": cookie})

	apptest.MustContain(t, spaces.Body, `"url":"`+"https://example.test"+`/api/coding"`, `"name":"read_file"`,
		`"readOnly":true`,
		`"id":"coding:read"`, `"id":"coding:approve"`, "Read files and search code")
}

func TestTheConnectorAddressFollowsConnectorURLNotTheAppAddress(t *testing.T) {
	t.Parallel()

	const connector = "https://connect.example.test"

	env := apptest.NewEnvWith(t, func(cfg *config.Config) { cfg.ConnectorURL = connector })
	cookie := apptest.Register(t, env.App, "Sari", "sari@example.com", "correct horse")

	spaces := apptest.Do(t, env.App, "GET", "/api/spaces", "", map[string]string{"Cookie": cookie})
	apptest.MustContain(t, spaces.Body, `"url":"`+connector+`/api/coding"`)
	if strings.Contains(string(spaces.Body), "https://example.test") {
		t.Fatal("connector response must not use the app address")
	}

	resource := apptest.Do(t, env.App, "GET", "/.well-known/oauth-protected-resource/api/coding", "", nil)
	apptest.MustContain(t, resource.Body, `"resource":"`+connector+`/api/coding"`,
		`"authorization_servers":["`+connector+`"]`)

	issuer := apptest.Do(t, env.App, "GET", "/.well-known/oauth-authorization-server", "", nil)
	apptest.MustContain(t, issuer.Body, `"issuer":"`+connector+`"`, connector+"/oauth/token")

	challenge := apptest.Do(t, env.App, "POST", "/api/coding", apptest.RPC("tools/list", "{}"), apptest.MCPHeaders(""))
	apptest.MustContain(t, []byte(challenge.Header.Get("WWW-Authenticate")),
		`resource_metadata="`+connector+`/.well-known/oauth-protected-resource/api/coding"`)
}
