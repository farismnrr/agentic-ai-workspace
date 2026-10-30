package projects_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
	"github.com/gofiber/fiber/v3"
)

type person struct {
	Cookie, token, dir string
}

var projectWorks sync.Map

func callProjectTool(t *testing.T, app *fiber.App, token, name, args string) apptest.Reply {
	t.Helper()
	value, ok := projectWorks.Load(token)
	if !ok {
		begin := apptest.CallTool(t, app, token, "begin_work", `{"role":"developer","summary":"Verify project isolation"}`)
		var response struct {
			Result struct {
				Structured struct {
					WorkID string `json:"work_id"`
				} `json:"structuredContent"`
			} `json:"result"`
		}
		if err := json.Unmarshal(begin.Body, &response); err != nil || response.Result.Structured.WorkID == "" {
			t.Fatalf("begin_work: %v %s", err, begin.Body)
		}
		value, _ = projectWorks.LoadOrStore(token, response.Result.Structured.WorkID)
	}
	if args == `{}` {
		args = `{"work_id":"` + value.(string) + `"}`
	} else {
		args = strings.Replace(args, "{", `{"work_id":"`+value.(string)+`",`, 1)
	}
	return apptest.CallTool(t, app, token, name, args)
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

	callProjectTool(t, app, sari.token, "write_file", `{"path":"secret.txt","content":"SARI ONLY"}`)

	// Bob sees an empty workspace of his own: neither Sari's new file nor her sample files.
	list := callProjectTool(t, app, bob.token, "list_files", `{}`)
	apptest.MustNotContain(t, list.Body, "secret.txt", "hello.txt")
	apptest.MustContain(t, callProjectTool(t, app, bob.token, "read_file", `{"path":"secret.txt"}`).Body,
		`"isError":true`)

	// Bob reaching for Sari's folder by name must fail too.
	escape := `{"path":"../` + filepath.Base(sari.dir) + `/secret.txt"}`
	probe := callProjectTool(t, app, bob.token, "read_file", escape)
	apptest.MustContain(t, probe.Body, `"isError":true`)
	apptest.MustNotContain(t, probe.Body, "SARI ONLY")

	// Bob writing the same path never touches Sari's file.
	callProjectTool(t, app, bob.token, "write_file", `{"path":"secret.txt","content":"BOB"}`)
	apptest.MustContain(t, callProjectTool(t, app, sari.token, "read_file", `{"path":"secret.txt"}`).Body, "SARI ONLY")

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
	callProjectTool(t, app, token, "list_files", `{}`)

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
