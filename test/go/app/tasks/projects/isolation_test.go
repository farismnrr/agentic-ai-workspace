package projects_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

type person struct {
	Cookie, token, dir string
}

func TestPeopleNeverShareAWorkspace(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	app := env.App
	sariCookie, clientID := apptest.SignedInClient(t, app)
	apptest.SeedWorkspace(t, env, sariCookie, "ChatGPT")
	sari := person{Cookie: sariCookie, token: apptest.Connect(t, app, sariCookie, clientID, "coding:read coding:write")}
	sari.dir = apptest.UserWorkspace(t, env, sari.Cookie, "ChatGPT")

	bobCookie := apptest.CookieOf(t, apptest.Do(t, app, "POST", "/api/auth/register",
		`{"name":"Bob","email":"bob@example.com","password":"`+apptest.TestPassword+`"}`, apptest.JSONHeaders()))
	bob := person{Cookie: bobCookie, token: apptest.Connect(t, app, bobCookie, clientID, "coding:read coding:write")}
	bob.dir = apptest.UserWorkspace(t, env, bob.Cookie, "ChatGPT")

	apptest.EnableTool(t, app, sari.Cookie, "write_file")
	apptest.EnableTool(t, app, bob.Cookie, "write_file")

	apptest.CallTool(t, app, sari.token, "write_file", `{"path":"secret.txt","content":"SARI ONLY"}`)

	// Bob sees an empty workspace of his own: neither Sari's new file nor her sample files.
	list := apptest.CallTool(t, app, bob.token, "list_files", `{}`)
	apptest.MustNotContain(t, list.Body, "secret.txt", "hello.txt")
	apptest.MustContain(t, apptest.CallTool(t, app, bob.token, "read_file", `{"path":"secret.txt"}`).Body,
		`"isError":true`)

	// Bob reaching for Sari's folder by name must fail too.
	escape := `{"path":"../` + filepath.Base(sari.dir) + `/secret.txt"}`
	probe := apptest.CallTool(t, app, bob.token, "read_file", escape)
	apptest.MustContain(t, probe.Body, `"isError":true`)
	apptest.MustNotContain(t, probe.Body, "SARI ONLY")

	// Bob writing the same path never touches Sari's file.
	apptest.CallTool(t, app, bob.token, "write_file", `{"path":"secret.txt","content":"BOB"}`)
	apptest.MustContain(t, apptest.CallTool(t, app, sari.token, "read_file", `{"path":"secret.txt"}`).Body, "SARI ONLY")

	if filepath.Dir(sari.dir) != filepath.Dir(bob.dir) || sari.dir == bob.dir {
		t.Fatalf("each person needs a distinct folder, got %q and %q", sari.dir, bob.dir)
	}
}

func TestDeletingAnAccountErasesItsWorkspace(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	app := env.App
	cookie, clientID := apptest.SignedInClient(t, app)
	token := apptest.Connect(t, app, cookie, clientID, "coding:read")
	apptest.CallTool(t, app, token, "list_files", `{}`)

	dir := apptest.UserWorkspace(t, env, cookie, "ChatGPT")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the workspace should exist before deletion: %v", err)
	}

	gone := apptest.Do(t, app, "POST", "/api/auth/delete", `{"password":"`+apptest.TestPassword+`"}`,
		map[string]string{"Cookie": cookie, "Content-Type": "application/json"})
	if gone.Status != 204 {
		t.Fatalf("account deletion failed: %d %s", gone.Status, gone.Body)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("the workspace must be deleted with the account, got %v", err)
	}
}
