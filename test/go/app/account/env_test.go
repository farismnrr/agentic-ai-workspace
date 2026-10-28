package account_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestProjectEnvIsScopedMaskedAndReplaceable(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t)
	created := apptest.Do(t, a.Environment.App, "POST", "/api/projects", `{"name":"Second"}`, apptest.JSONCookie(a.Cookie))
	projectB := created.JSON(t)["id"].(string) //nolint:forcetypeassert // API test

	secret := putEnv(t, a, a.Project, `{"name":"API_TOKEN","value":"first secret"}`)
	if secret.Status != 200 || strings.Contains(string(secret.Body), "first secret") {
		t.Fatalf("secret save response leaks the value: %d %s", secret.Status, secret.Body)
	}
	putEnv(t, a, a.Project, `{"name":"PUBLIC_URL","value":"https://example.test","isSecret":false}`)

	listed := getEnv(t, a, a.Project)
	apptest.MustContain(t, listed.Body, `"name":"API_TOKEN"`, `"isSecret":true`, `"name":"PUBLIC_URL"`,
		`"value":"https://example.test"`)
	if strings.Contains(string(listed.Body), "first secret") {
		t.Fatalf("list response leaks a secret: %s", listed.Body)
	}
	if other := getEnv(t, a, projectB); other.Status != 200 || string(other.Body) != "[]" {
		t.Fatalf("second project received variables: %d %s", other.Status, other.Body)
	}

	replaced := putEnv(t, a, a.Project, `{"name":"API_TOKEN","value":"replacement"}`)
	if replaced.Status != 200 || strings.Contains(string(replaced.Body), "replacement") {
		t.Fatalf("secret replacement response leaked the value: %d %s", replaced.Status, replaced.Body)
	}
	deleted := apptest.Do(t, a.Environment.App, "DELETE", "/api/projects/"+a.Project+"/env/PUBLIC_URL", "",
		apptest.JSONCookie(a.Cookie))
	if deleted.Status != 204 {
		t.Fatalf("delete status = %d, want 204", deleted.Status)
	}
}

func TestProjectEnvRejectsInvalidNamesAndEnforcesLimit(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t)
	for _, name := range []string{"PATH", "GOFLAGS", "bad-name"} {
		response := putEnv(t, a, a.Project, `{"name":"`+name+`","value":"x"}`)
		if response.Status != 400 {
			t.Errorf("name %q status = %d, want 400", name, response.Status)
		}
	}
	for i := 0; i < 50; i++ {
		name := "VAR_" + strings.Repeat("0", 2) + string(rune('A'+i/26)) + string(rune('A'+i%26))
		if response := putEnv(t, a, a.Project, `{"name":"`+name+`","value":"x"}`); response.Status != 200 {
			t.Fatalf("save %s = %d %s", name, response.Status, response.Body)
		}
	}
	if response := putEnv(t, a, a.Project, `{"name":"EXTRA","value":"x"}`); response.Status != 409 {
		t.Fatalf("51st variable status = %d, want 409", response.Status)
	}
	if response := putEnv(t, a, a.Project, `{"name":"VAR_00AA","value":"replace"}`); response.Status != 200 {
		t.Fatalf("replace at limit status = %d, want 200", response.Status)
	}
}

func TestProjectEnvIsHiddenFromOtherOwners(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t)
	other := apptest.CookieOf(t, apptest.Do(t, a.Environment.App, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, apptest.JSONHeaders()))

	if got := apptest.Do(t, a.Environment.App, "GET", "/api/projects/"+a.Project+"/env", "",
		apptest.JSONCookie(other)).Status; got != 404 {
		t.Fatalf("other owner list status = %d, want 404", got)
	}
	if got := apptest.Do(t, a.Environment.App, "PUT", "/api/projects/"+a.Project+"/env",
		`{"name":"TOKEN","value":"secret"}`, apptest.JSONCookie(other)).Status; got != 404 {
		t.Fatalf("other owner put status = %d, want 404", got)
	}
}

func putEnv(t *testing.T, a apptest.AgentRig, project, body string) apptest.Reply {
	t.Helper()
	return apptest.Do(t, a.Environment.App, "PUT", "/api/projects/"+project+"/env", body, apptest.JSONCookie(a.Cookie))
}

func getEnv(t *testing.T, a apptest.AgentRig, project string) apptest.Reply {
	t.Helper()
	return apptest.Do(t, a.Environment.App, "GET", "/api/projects/"+project+"/env", "", apptest.JSONCookie(a.Cookie))
}
