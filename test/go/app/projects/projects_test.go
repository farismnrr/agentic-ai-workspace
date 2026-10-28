package projects_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestProjectLifecycleWithItsOwnFolder(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie := apptest.Register(t, env.App, "Sari", "sari@example.com", "correct horse")

	body := `{"name":"  Bakery site ","description":"Menu and orders"}`
	created := apptest.Do(t, env.App, "POST", "/api/projects", body, map[string]string{
		"Cookie":       cookie,
		"Content-Type": "application/json",
	})
	if created.Status != 201 {
		t.Fatalf("create failed: %d %s", created.Status, created.Body)
	}

	id, _ := created.JSON(t)["id"].(string)
	if info, err := os.Stat(filepath.Join(env.DataDir, "projects", id)); err != nil || !info.IsDir() {
		t.Fatalf("the project needs its own folder: %v", err)
	}

	apptest.MustContain(t, created.Body, `"name":"Bakery site"`)
	apptest.MustContain(t, apptest.Do(t, env.App, "GET", "/api/projects", "", map[string]string{
		"Cookie":       cookie,
		"Content-Type": "application/json",
	}).Body, id)

	renamed := apptest.Do(t, env.App, "PUT", "/api/projects/"+id, `{"name":"Bakery","description":"New"}`,
		map[string]string{"Cookie": cookie, "Content-Type": "application/json"})
	apptest.MustContain(t, renamed.Body, `"name":"Bakery"`, `"description":"New"`)

	if got := apptest.Do(t, env.App, "DELETE", "/api/projects/"+id, "", map[string]string{
		"Cookie":       cookie,
		"Content-Type": "application/json",
	}).Status; got != 204 {
		t.Fatalf("archive failed: %d", got)
	}

	if list := apptest.Do(t, env.App, "GET", "/api/projects", "", map[string]string{
		"Cookie":       cookie,
		"Content-Type": "application/json",
	}); strings.Contains(string(list.Body), id) {
		t.Fatalf("archived projects leave the list: %s", list.Body)
	}
}

func TestProjectsAreValidatedAndPrivate(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	cookie := apptest.Register(t, env.App, "Sari", "sari@example.com", "correct horse")

	if got := apptest.Do(t, env.App, "POST", "/api/projects", `{"name":" "}`, map[string]string{
		"Cookie":       cookie,
		"Content-Type": "application/json",
	}).Status; got != 400 {
		t.Fatalf("a blank name must be rejected, got %d", got)
	}

	id, _ := apptest.Do(t, env.App, "POST", "/api/projects", `{"name":"Mine"}`, map[string]string{
		"Cookie":       cookie,
		"Content-Type": "application/json",
	}).JSON(t)["id"].(string)
	other := apptest.Register(t, env.App, "Eve", "eve@example.com", "correct horse")

	if got := apptest.Do(t, env.App, "GET", "/api/projects/"+id, "",
		map[string]string{"Cookie": other}).Status; got != 404 {
		t.Fatalf("another person's project must be hidden, got %d", got)
	}

	if got := apptest.Do(t, env.App, "GET", "/api/projects", "", nil).Status; got != 401 {
		t.Fatalf("anonymous requests must be refused, got %d", got)
	}
}
