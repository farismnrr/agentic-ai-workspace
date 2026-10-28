package tools_test

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
	"github.com/gofiber/fiber/v3"
)

var toolWorks sync.Map

func callTool(t *testing.T, app *fiber.App, token, name, args string) apptest.Reply {
	t.Helper()
	if name != "begin_work" {
		value, ok := toolWorks.Load(token)
		if !ok {
			begin := apptest.CallTool(t, app, token, "begin_work", `{"role":"developer","summary":"Exercise tool access controls"}`)
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
			value, _ = toolWorks.LoadOrStore(token, response.Result.Structured.WorkID)
		}
		if args == `{}` {
			args = `{"work_id":"` + value.(string) + `"}`
		} else {
			args = strings.Replace(args, "{", `{"work_id":"`+value.(string)+`",`, 1)
		}
	}
	return apptest.CallTool(t, app, token, name, args)
}

func enableTool(t *testing.T, app *fiber.App, cookie, tool string) {
	t.Helper()

	apptest.EnableTool(t, app, cookie, tool)
}

type activityRow struct {
	ID      string `json:"id"`
	Tool    string `json:"tool"`
	Outcome string `json:"outcome"`
	CanUndo bool   `json:"canUndo"`
}

func activity(t *testing.T, app *fiber.App, cookie string) []activityRow {
	t.Helper()

	var rows []activityRow
	body := apptest.Do(t, app, "GET", "/api/activity", "", map[string]string{"Cookie": cookie}).Body
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatal(err)
	}

	return rows
}

func undo(t *testing.T, app *fiber.App, cookie, id string) int {
	t.Helper()

	return apptest.Do(t, app, "POST", "/api/activity/"+id+"/undo", "", map[string]string{"Cookie": cookie}).Status
}

func TestToolSwitchesStartOffForWritingAndRunning(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	app := env.App
	cookie, _ := apptest.SignedInClient(t, app)

	var spaces []struct {
		Tools []struct {
			Name    string `json:"name"`
			Group   string `json:"group"`
			Enabled bool   `json:"enabled"`
		} `json:"tools"`
	}

	body := apptest.Do(t, app, "GET", "/api/spaces", "", map[string]string{"Cookie": cookie}).Body
	if err := json.Unmarshal(body, &spaces); err != nil || len(spaces) != 1 {
		t.Fatalf("one space expected: %v %s", err, body)
	}

	for _, tool := range spaces[0].Tools {
		want := tool.Group == "read" && tool.Name != "approve_plan"
		if tool.Enabled != want {
			t.Errorf("%s (%s) enabled=%v, want %v", tool.Name, tool.Group, tool.Enabled, want)
		}
	}
}

func TestWriteToolRefusesUntilSwitchedOn(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	app := env.App
	cookie, clientID := apptest.SignedInClient(t, app)
	apptest.SeedWorkspace(t, env, cookie, "ChatGPT")
	root := apptest.UserWorkspace(t, env, cookie, "ChatGPT")
	token := apptest.Connect(t, app, cookie, clientID, "coding:read coding:write")
	args := `{"path":"hello.txt","old_text":"hello","new_text":"howdy"}`

	off := callTool(t, app, token, "edit_file", args)
	apptest.MustContain(t, off.Body, `"isError":true`, "turned off")

	enableTool(t, app, cookie, "edit_file")
	apptest.MustContain(t, callTool(t, app, token, "edit_file", args).Body, "replaced 1 occurrence")

	if data := readFile(t, root, "hello.txt"); !strings.HasPrefix(data, "howdy") {
		t.Fatalf("the file should be edited, got %q", data)
	}

	if rows := activity(t, app, cookie); len(rows) < 2 || rows[0].Outcome != "ok" || rows[1].Outcome != "off" {
		t.Fatalf("both the refusal and the edit must be logged, newest first: %+v", rows)
	}
}

func TestMissingScopeAsksForStepUpSignIn(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	app := env.App
	cookie, clientID := apptest.SignedInClient(t, app)
	token := apptest.Connect(t, app, cookie, clientID, "coding:read")

	enableTool(t, app, cookie, "write_file")

	reply := callTool(t, app, token, "write_file", `{"path":"x.txt","content":"x"}`)
	apptest.MustContain(t, reply.Body, `"isError":true`, "mcp/www_authenticate", "insufficient_scope", "coding:write",
		"resource_metadata")
}

func TestTrackedEditUndoLivesInTaskChanges(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	app := env.App
	cookie, clientID := apptest.SignedInClient(t, app)
	apptest.SeedWorkspace(t, env, cookie, "ChatGPT")
	root := apptest.UserWorkspace(t, env, cookie, "ChatGPT")
	token := apptest.Connect(t, app, cookie, clientID, "coding:read coding:write")
	enableTool(t, app, cookie, "edit_file")
	callTool(t, app, token, "edit_file", `{"path":"hello.txt","old_text":"hello","new_text":"howdy"}`)

	row := activity(t, app, cookie)[0]
	// Tracked edits are undoable from the task Changes tab, not from the legacy activity feed.
	if row.CanUndo || undo(t, app, cookie, row.ID) == 204 {
		t.Fatalf("tracked edits must use the task Changes tab for undo: %+v", row)
	}
	if data := readFile(t, root, "hello.txt"); data != "howdy from the workspace\n" {
		t.Fatalf("the refused activity undo must leave the edit intact, got %q", data)
	}
}

func TestUndoIsRefusedAfterALaterChangeAndForOtherPeople(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	app := env.App
	cookie, clientID := apptest.SignedInClient(t, app)
	token := apptest.Connect(t, app, cookie, clientID, "coding:read coding:write")
	enableTool(t, app, cookie, "edit_file")
	callTool(t, app, token, "edit_file", `{"path":"hello.txt","old_text":"hello","new_text":"one"}`)
	first := activity(t, app, cookie)[0]
	callTool(t, app, token, "edit_file", `{"path":"hello.txt","old_text":"one","new_text":"two"}`)

	if got := undo(t, app, cookie, first.ID); got != 409 {
		t.Fatalf("a stale undo must be refused, got %d", got)
	}

	other := apptest.CookieOf(t, apptest.Do(t, app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, apptest.JSONHeaders()))
	if got := undo(t, app, other, first.ID); got != 404 {
		t.Fatalf("someone else's entry must not be reachable, got %d", got)
	}
}

func TestTasksRunOnlyWhenSwitchedOnAndNeverExposeTheirFile(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	app := env.App
	cookie, clientID := apptest.SignedInClient(t, app)
	apptest.SeedWorkspace(t, env, cookie, "ChatGPT")
	token := apptest.Connect(t, app, cookie, clientID, "coding:read coding:exec")

	apptest.MustContain(t, callTool(t, app, token, "list_tasks", `{}`).Body, "echo", "Say hello")
	apptest.MustContain(t, callTool(t, app, token, "run_task", `{"name":"echo"}`).Body, "turned off")

	apptest.RequireDocker(t)
	enableTool(t, app, cookie, "run_task")
	apptest.MustContain(t, callTool(t, app, token, "run_task", `{"name":"echo"}`).Body, "hello task", `"exit_code":0`)
	apptest.MustContain(t, callTool(t, app, token, "run_task", `{"name":"nope"}`).Body, "unknown task")
	apptest.MustContain(t, callTool(t, app, token, "read_file", `{"path":".workspace/tasks.json"}`).Body, "not available")
}

func readFile(t *testing.T, root, name string) string {
	t.Helper()
	directory, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = directory.Close() }()
	data, err := directory.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
