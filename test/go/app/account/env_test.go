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

func TestProjectEnvRevealRequiresCurrentPasswordAndReturnsOnlyOneSecret(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t)
	putEnv(t, a, a.Project, `{"name":"API_TOKEN","value":"private-token"}`)
	putEnv(t, a, a.Project, `{"name":"OTHER_TOKEN","value":"other-private-token"}`)
	path := "/api/projects/" + a.Project + "/env/API_TOKEN/reveal"

	wrong := apptest.Do(t, a.Environment.App, "POST", path, `{"password":"incorrect password"}`,
		apptest.JSONCookie(a.Cookie))
	if wrong.Status != 401 || strings.Contains(string(wrong.Body), "private-token") {
		t.Fatalf("wrong password reveal = %d %s", wrong.Status, wrong.Body)
	}
	assertRevealNoStore(t, wrong)
	correct := apptest.Do(t, a.Environment.App, "POST", path, `{"password":"correct horse"}`,
		apptest.JSONCookie(a.Cookie))
	if correct.Status != 200 || string(correct.Body) != `{"name":"API_TOKEN","value":"private-token"}` {
		t.Fatalf("correct password reveal = %d %s", correct.Status, correct.Body)
	}
	assertRevealNoStore(t, correct)
	if !strings.Contains(string(getEnv(t, a, a.Project).Body), `"isSecret":true`) ||
		strings.Contains(string(getEnv(t, a, a.Project).Body), "private-token") {
		t.Fatal("environment list revealed secret plaintext")
	}
}

func TestProjectEnvRevealRejectsCrossSiteAndLimitsFailedPasswords(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t)
	putEnv(t, a, a.Project, `{"name":"API_TOKEN","value":"private-token"}`)
	path := "/api/projects/" + a.Project + "/env/API_TOKEN/reveal"
	crossSite := apptest.Do(t, a.Environment.App, "POST", path, `{"password":"correct horse"}`,
		map[string]string{"Content-Type": "application/json", "Cookie": a.Cookie, "Origin": "https://attacker.example"})
	if crossSite.Status != 403 || strings.Contains(string(crossSite.Body), "private-token") {
		t.Fatalf("cross-site reveal = %d %s", crossSite.Status, crossSite.Body)
	}
	assertRevealNoStore(t, crossSite)
	for attempt := 0; attempt < 5; attempt++ {
		response := apptest.Do(t, a.Environment.App, "POST", path, `{"password":"incorrect password"}`,
			apptest.JSONCookie(a.Cookie))
		if response.Status != 401 {
			t.Fatalf("failed password attempt %d = %d %s", attempt+1, response.Status, response.Body)
		}
	}
	limited := apptest.Do(t, a.Environment.App, "POST", path, `{"password":"correct horse"}`,
		apptest.JSONCookie(a.Cookie))
	if limited.Status != 429 || strings.Contains(string(limited.Body), "private-token") {
		t.Fatalf("limited reveal = %d %s", limited.Status, limited.Body)
	}
	assertRevealNoStore(t, limited)

	anonymous := apptest.Do(t, a.Environment.App, "POST", path, `{"password":"correct horse"}`, apptest.JSONHeaders())
	if anonymous.Status != 401 {
		t.Fatalf("anonymous reveal = %d %s", anonymous.Status, anonymous.Body)
	}
	assertRevealNoStore(t, anonymous)
}

func TestProjectEnvRevealNoStoreForWrongOwnerMethodAndOversizedBody(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t)
	putEnv(t, a, a.Project, `{"name":"API_TOKEN","value":"private-token"}`)
	path := "/api/projects/" + a.Project + "/env/API_TOKEN/reveal"
	other := apptest.CookieOf(t, apptest.Do(t, a.Environment.App, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, apptest.JSONHeaders()))
	wrongOwner := apptest.Do(t, a.Environment.App, "POST", path, `{"password":"correct horse"}`,
		apptest.JSONCookie(other))
	if wrongOwner.Status != 404 || strings.Contains(string(wrongOwner.Body), "private-token") {
		t.Fatalf("wrong-owner reveal = %d %s", wrongOwner.Status, wrongOwner.Body)
	}
	assertRevealNoStore(t, wrongOwner)

	wrongMethod := apptest.Do(t, a.Environment.App, "GET", path, "", apptest.JSONCookie(a.Cookie))
	if wrongMethod.Status < 400 || strings.Contains(string(wrongMethod.Body), "private-token") {
		t.Fatalf("wrong-method reveal = %d %s", wrongMethod.Status, wrongMethod.Body)
	}
	assertRevealNoStore(t, wrongMethod)

	oversized := apptest.Do(t, a.Environment.App, "POST", path, `{"password":"`+strings.Repeat("x", 2048)+`"}`,
		apptest.JSONCookie(a.Cookie))
	if oversized.Status != 413 || strings.Contains(string(oversized.Body), "private-token") {
		t.Fatalf("oversized reveal = %d %s", oversized.Status, oversized.Body)
	}
	assertRevealNoStore(t, oversized)
}

func assertRevealNoStore(t *testing.T, response apptest.Reply) {
	t.Helper()
	if response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("Pragma") != "no-cache" {
		t.Fatalf("reveal response is cacheable: %v", response.Header)
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
