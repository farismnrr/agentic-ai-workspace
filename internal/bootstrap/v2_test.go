package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func callTool(t *testing.T, app *fiber.App, token, name, args string) reply {
	t.Helper()

	call := rpc("tools/call", `{"name":"`+name+`","arguments":`+args+`}`)

	return do(t, app, "POST", "/api/coding", call, mcpHeaders(token))
}

func enableTool(t *testing.T, app *fiber.App, cookie, tool string) {
	t.Helper()

	const body = `{"enabled":true}`
	headers := map[string]string{"Cookie": cookie, "Content-Type": "application/json"}

	if got := do(t, app, "PUT", "/api/spaces/coding/tools/"+tool, body, headers).Status; got != 204 {
		t.Fatalf("enabling %s failed: %d", tool, got)
	}
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
	body := do(t, app, "GET", "/api/activity", "", map[string]string{"Cookie": cookie}).Body
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatal(err)
	}

	return rows
}

func undo(t *testing.T, app *fiber.App, cookie, id string) int {
	t.Helper()

	return do(t, app, "POST", "/api/activity/"+id+"/undo", "", map[string]string{"Cookie": cookie}).Status
}

func TestToolSwitchesStartOffForWritingAndRunning(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, _ := signedInClient(t, app)

	var spaces []struct {
		Slug  string `json:"slug"`
		Tools []struct {
			Name    string `json:"name"`
			Group   string `json:"group"`
			Enabled bool   `json:"enabled"`
		} `json:"tools"`
	}

	body := do(t, app, "GET", "/api/spaces", "", map[string]string{"Cookie": cookie}).Body
	if err := json.Unmarshal(body, &spaces); err != nil || len(spaces) != 2 {
		t.Fatalf("Coding and Orchestrator spaces expected: %v %s", err, body)
	}

	for _, space := range spaces {
		for _, tool := range space.Tools {
			if want := tool.Group == "read"; tool.Enabled != want {
				t.Errorf("%s/%s (%s) enabled=%v, want %v", space.Slug, tool.Name, tool.Group, tool.Enabled, want)
			}
		}
	}
}

func TestWriteToolRefusesUntilSwitchedOn(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	root := userWorkspace(t, app, cookie, "ChatGPT")
	token := connect(t, app, cookie, clientID, "coding:read coding:write")
	args := `{"path":"hello.txt","old_text":"hello","new_text":"howdy"}`

	off := callTool(t, app, token, "edit_file", args)
	mustContain(t, off.Body, `"isError":true`, "turned off")

	enableTool(t, app, cookie, "edit_file")
	mustContain(t, callTool(t, app, token, "edit_file", args).Body, "replaced 1 occurrence")

	if data := fileText(t, root, "hello.txt"); !strings.HasPrefix(data, "howdy") {
		t.Fatalf("the file should be edited, got %q", data)
	}

	if rows := activity(t, app, cookie); len(rows) < 2 || rows[0].Outcome != "ok" || rows[1].Outcome != "off" {
		t.Fatalf("both the refusal and the edit must be logged, newest first: %+v", rows)
	}
}

func TestMissingScopeAsksForStepUpSignIn(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	token := connect(t, app, cookie, clientID, "coding:read")

	enableTool(t, app, cookie, "write_file")

	reply := callTool(t, app, token, "write_file", `{"path":"x.txt","content":"x"}`)
	mustContain(t, reply.Body, `"isError":true`, "mcp/www_authenticate", "insufficient_scope", "coding:write",
		"resource_metadata")
}

func TestUndoRestoresTheFileOnce(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	root := userWorkspace(t, app, cookie, "ChatGPT")
	token := connect(t, app, cookie, clientID, "coding:read coding:write")
	enableTool(t, app, cookie, "edit_file")
	callTool(t, app, token, "edit_file", `{"path":"hello.txt","old_text":"hello","new_text":"howdy"}`)

	row := activity(t, app, cookie)[0]
	if !row.CanUndo || undo(t, app, cookie, row.ID) != 204 {
		t.Fatalf("a fresh edit must be undoable: %+v", row)
	}

	if data := fileText(t, root, "hello.txt"); data != "hello from the workspace\n" {
		t.Fatalf("original content should be back, got %q", data)
	}

	if got := undo(t, app, cookie, row.ID); got != 409 {
		t.Fatalf("a second undo must be refused, got %d", got)
	}
}

func TestUndoIsRefusedAfterALaterChangeAndForOtherPeople(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	token := connect(t, app, cookie, clientID, "coding:read coding:write")
	enableTool(t, app, cookie, "edit_file")
	callTool(t, app, token, "edit_file", `{"path":"hello.txt","old_text":"hello","new_text":"one"}`)
	first := activity(t, app, cookie)[0]
	callTool(t, app, token, "edit_file", `{"path":"hello.txt","old_text":"one","new_text":"two"}`)

	if got := undo(t, app, cookie, first.ID); got != 409 {
		t.Fatalf("a stale undo must be refused, got %d", got)
	}

	other := cookieOf(t, do(t, app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, jsonHeaders()))
	if got := undo(t, app, other, first.ID); got != 404 {
		t.Fatalf("someone else's entry must not be reachable, got %d", got)
	}
}

func TestTasksRunOnlyWhenSwitchedOnAndNeverExposeTheirFile(t *testing.T) {
	t.Parallel()
	requireDocker(t)

	app := newApp(t)
	cookie, clientID := signedInClient(t, app)
	token := connect(t, app, cookie, clientID, "coding:read coding:exec")

	mustContain(t, callTool(t, app, token, "list_tasks", `{}`).Body, "echo", "Say hello")
	mustContain(t, callTool(t, app, token, "run_task", `{"name":"echo"}`).Body, "turned off")

	enableTool(t, app, cookie, "run_task")
	mustContain(t, callTool(t, app, token, "run_task", `{"name":"echo"}`).Body, "hello task", `"exit_code":0`)
	mustContain(t, callTool(t, app, token, "run_task", `{"name":"nope"}`).Body, "unknown task")
	mustContain(t, callTool(t, app, token, "read_file", `{"path":".workspace/tasks.json"}`).Body, "not available")
}
