package bootstrap

import (
	"strings"
	"testing"
)

func TestProjectEnvIsScopedMaskedAndReplaceable(t *testing.T) {
	t.Parallel()
	a := newAgentRig(t)
	created := do(t, a.env.app, "POST", "/api/projects", `{"name":"Second"}`, jsonCookie(a.cookie))
	projectB := created.JSON(t)["id"].(string) //nolint:forcetypeassert // API test

	secret := putEnv(t, a, a.project, `{"name":"API_TOKEN","value":"first secret"}`)
	if secret.Status != 200 || strings.Contains(string(secret.Body), "first secret") {
		t.Fatalf("secret save response leaks the value: %d %s", secret.Status, secret.Body)
	}
	putEnv(t, a, a.project, `{"name":"PUBLIC_URL","value":"https://example.test","isSecret":false}`)

	listed := getEnv(t, a, a.project)
	mustContain(t, listed.Body, `"name":"API_TOKEN"`, `"isSecret":true`, `"name":"PUBLIC_URL"`,
		`"value":"https://example.test"`)
	if strings.Contains(string(listed.Body), "first secret") {
		t.Fatalf("list response leaks a secret: %s", listed.Body)
	}
	if other := getEnv(t, a, projectB); other.Status != 200 || string(other.Body) != "[]" {
		t.Fatalf("second project received variables: %d %s", other.Status, other.Body)
	}

	replaced := putEnv(t, a, a.project, `{"name":"API_TOKEN","value":"replacement"}`)
	if replaced.Status != 200 || strings.Contains(string(replaced.Body), "replacement") {
		t.Fatalf("secret replacement response leaked the value: %d %s", replaced.Status, replaced.Body)
	}
	deleted := do(t, a.env.app, "DELETE", "/api/projects/"+a.project+"/env/PUBLIC_URL", "", jsonCookie(a.cookie))
	if deleted.Status != 204 {
		t.Fatalf("delete status = %d, want 204", deleted.Status)
	}
}

func TestProjectEnvRejectsInvalidNamesAndEnforcesLimit(t *testing.T) {
	t.Parallel()
	a := newAgentRig(t)
	for _, name := range []string{"PATH", "GOFLAGS", "bad-name"} {
		response := putEnv(t, a, a.project, `{"name":"`+name+`","value":"x"}`)
		if response.Status != 400 {
			t.Errorf("name %q status = %d, want 400", name, response.Status)
		}
	}
	for i := 0; i < 50; i++ {
		name := "VAR_" + strings.Repeat("0", 2) + string(rune('A'+i/26)) + string(rune('A'+i%26))
		if response := putEnv(t, a, a.project, `{"name":"`+name+`","value":"x"}`); response.Status != 200 {
			t.Fatalf("save %s = %d %s", name, response.Status, response.Body)
		}
	}
	if response := putEnv(t, a, a.project, `{"name":"EXTRA","value":"x"}`); response.Status != 409 {
		t.Fatalf("51st variable status = %d, want 409", response.Status)
	}
	if response := putEnv(t, a, a.project, `{"name":"VAR_00AA","value":"replace"}`); response.Status != 200 {
		t.Fatalf("replace at limit status = %d, want 200", response.Status)
	}
}

func TestProjectEnvIsHiddenFromOtherOwners(t *testing.T) {
	t.Parallel()
	a := newAgentRig(t)
	other := cookieOf(t, do(t, a.env.app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, jsonHeaders()))

	if got := do(t, a.env.app, "GET", "/api/projects/"+a.project+"/env", "", jsonCookie(other)).Status; got != 404 {
		t.Fatalf("other owner list status = %d, want 404", got)
	}
	if got := do(t, a.env.app, "PUT", "/api/projects/"+a.project+"/env",
		`{"name":"TOKEN","value":"secret"}`, jsonCookie(other)).Status; got != 404 {
		t.Fatalf("other owner put status = %d, want 404", got)
	}
}

func putEnv(t *testing.T, a agentRig, project, body string) reply {
	t.Helper()
	return do(t, a.env.app, "PUT", "/api/projects/"+project+"/env", body, jsonCookie(a.cookie))
}

func getEnv(t *testing.T, a agentRig, project string) reply {
	t.Helper()
	return do(t, a.env.app, "GET", "/api/projects/"+project+"/env", "", jsonCookie(a.cookie))
}
