package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectLifecycleWithItsOwnFolder(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)

	body := `{"name":"  Bakery site ","description":"Menu and orders"}`
	created := do(t, env.app, "POST", "/api/projects", body, jsonCookie(cookie))
	if created.Status != 201 {
		t.Fatalf("create failed: %d %s", created.Status, created.Body)
	}

	id, _ := created.JSON(t)["id"].(string)
	if info, err := os.Stat(filepath.Join(env.dataDir, "projects", id)); err != nil || !info.IsDir() {
		t.Fatalf("the project needs its own folder: %v", err)
	}

	mustContain(t, created.Body, `"name":"Bakery site"`)
	mustContain(t, do(t, env.app, "GET", "/api/projects", "", jsonCookie(cookie)).Body, id)

	renamed := do(t, env.app, "PUT", "/api/projects/"+id, `{"name":"Bakery","description":"New"}`, jsonCookie(cookie))
	mustContain(t, renamed.Body, `"name":"Bakery"`, `"description":"New"`)

	if got := do(t, env.app, "DELETE", "/api/projects/"+id, "", jsonCookie(cookie)).Status; got != 204 {
		t.Fatalf("archive failed: %d", got)
	}

	if list := do(t, env.app, "GET", "/api/projects", "", jsonCookie(cookie)); strings.Contains(string(list.Body), id) {
		t.Fatalf("archived projects leave the list: %s", list.Body)
	}
}

func TestProjectsAreValidatedAndPrivate(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)

	if got := do(t, env.app, "POST", "/api/projects", `{"name":" "}`, jsonCookie(cookie)).Status; got != 400 {
		t.Fatalf("a blank name must be rejected, got %d", got)
	}

	id, _ := do(t, env.app, "POST", "/api/projects", `{"name":"Mine"}`, jsonCookie(cookie)).JSON(t)["id"].(string)
	other := cookieOf(t, do(t, env.app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, jsonHeaders()))

	if got := do(t, env.app, "GET", "/api/projects/"+id, "", jsonCookie(other)).Status; got != 404 {
		t.Fatalf("another person's project must be hidden, got %d", got)
	}

	if got := do(t, env.app, "GET", "/api/projects", "", nil).Status; got != 401 {
		t.Fatalf("anonymous requests must be refused, got %d", got)
	}
}

func TestStartingFromATemplateNeedsAKnownNameAndTheOwner(t *testing.T) {
	t.Parallel()

	a := rigOn(t, newEnv(t))
	path := "/api/projects/" + a.project + "/template"

	for _, bad := range []string{`{"template":""}`, `{"template":"../etc"}`, `{"template":"nope; rm -rf /"}`} {
		if got := do(t, a.env.app, "POST", path, bad, jsonCookie(a.cookie)).Status; got != 400 {
			t.Errorf("%s must be refused as an unknown template, got %d", bad, got)
		}
	}

	other := cookieOf(t, do(t, a.env.app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"`+testPassword+`"}`, jsonHeaders()))
	if got := do(t, a.env.app, "POST", path, `{"template":"landing"}`, jsonCookie(other)).Status; got != 404 {
		t.Errorf("someone else's project is not found: %d", got)
	}

	if err := os.WriteFile(filepath.Join(a.env.dataDir, "projects", a.project, "app"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := do(t, a.env.app, "POST", path, `{"template":"landing"}`, jsonCookie(a.cookie)).Status; got != 409 {
		t.Errorf("a project that already has an app/ is not started from a template: %d", got)
	}
}
