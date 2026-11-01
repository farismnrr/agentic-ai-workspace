package connectors_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

// Exercise the same MCP and owner HTTP boundaries used by connected clients.
// A connected assistant must not grant itself Deep Review by selecting Security.
func TestExternalSecurityRequiresOwnerEnabledDeepReview(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, client, "coding:read coding:exec")
	denied := func(tool, args string) {
		t.Helper()
		response := apptest.CallTool(t, env.App, token, tool, args)
		if !strings.Contains(string(response.Body), `"isError":true`) {
			t.Fatalf("%s unexpectedly allowed before owner opt-in: %s", tool, response.Body)
		}
	}

	denied("begin_work", `{"summary":"@deep-review should not grant privilege","role":"security","new_task":true}`)
	denied("begin_work", `{"summary":"Try to plan an elevated review","role":"developer","planned_roles":["security"],"new_task":true}`)

	started := mcpBegin(t, &env, token, `{"summary":"Review external changes","role":"researcher"}`)
	workID := started.Result.Structured.WorkID
	taskID := started.Result.Structured.TaskID
	headers := apptest.JSONCookie(cookie)
	taskURL := "/api/tasks/" + taskID
	if mode := apptest.Do(t, env.App, "GET", taskURL, "", headers).JSON(t)["mode"]; mode != "swift" {
		t.Fatalf("external work must start Swift: %v", mode)
	}
	denied("set_role", `{"work_id":"`+workID+`","role":"security"}`)
	denied("begin_work", `{"summary":"Review external changes","role":"security"}`)
	allowedRead := apptest.CallTool(t, env.App, token, "list_files", `{"work_id":"`+workID+`"}`)
	if strings.Contains(string(allowedRead.Body), `"isError":true`) {
		t.Fatalf("normal researcher should still work: %s", allowedRead.Body)
	}

	other := apptest.CookieOf(t, apptest.Do(t, env.App, "POST", "/api/auth/register",
		`{"name":"Another owner","email":"another-review@example.com","password":"correct horse"}`,
		apptest.JSONHeaders()))
	foreign := apptest.Do(t, env.App, "POST", taskURL+"/deep-review", "{}", apptest.JSONCookie(other))
	if foreign.Status != 404 {
		t.Fatalf("other owner's opt-in not denied: %d %s", foreign.Status, foreign.Body)
	}

	csrf := apptest.JSONCookie(cookie)
	csrf["Origin"] = "https://untrusted.invalid"
	forged := apptest.Do(t, env.App, "POST", taskURL+"/deep-review", "{}", csrf)
	if forged.Status != 403 {
		t.Fatalf("cross-site owner opt-in unexpectedly accepted: %d %s", forged.Status, forged.Body)
	}
	approved := apptest.Do(t, env.App, "POST", taskURL+"/deep-review", "{}", headers)
	if approved.Status != 200 || approved.JSON(t)["mode"] != "deep-review" {
		var source, mode, taskWork, state string
		var archived any
		_ = env.DB.QueryRow("SELECT source, review_mode, work_id, archived_at FROM tasks WHERE id = ?", taskID).
			Scan(&source, &mode, &taskWork, &archived)
		_ = env.DB.QueryRow("SELECT state FROM external_works WHERE task_id = ?", taskID).Scan(&state)
		t.Fatalf("owner opt-in failed: %d %s (source=%q mode=%q work=%q state=%q archived=%v)",
			approved.Status, approved.Body, source, mode, taskWork, state, archived)
	}
	repeat := apptest.Do(t, env.App, "POST", taskURL+"/deep-review", "{}", headers)
	if repeat.Status != 200 || repeat.JSON(t)["mode"] != "deep-review" {
		t.Fatalf("owner opt-in is not idempotent: %d %s", repeat.Status, repeat.Body)
	}

	switched := apptest.CallTool(t, env.App, token, "set_role",
		`{"work_id":"`+workID+`","role":"security"}`)
	if strings.Contains(string(switched.Body), `"isError":true`) ||
		!strings.Contains(string(switched.Body), `"changed":true`) {
		t.Fatalf("Security remained blocked after owner opt-in: %s", switched.Body)
	}
	canRead := apptest.CallTool(t, env.App, token, "list_files",
		`{"work_id":"`+workID+`"}`)
	if strings.Contains(string(canRead.Body), `"isError":true`) {
		t.Fatalf("Security cannot read project after owner opt-in: %s", canRead.Body)
	}
	denied("write_file", `{"work_id":"`+workID+`","path":"bad.txt","content":"no"}`)
	denied("begin_work", `{"summary":"New task cannot inherit a prior opt-in","role":"security","new_task":true}`)
}
