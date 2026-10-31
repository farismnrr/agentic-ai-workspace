package molab_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestChatGPTMoLabRequiresDeveloperWork(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, clientID := apptest.SignedInClient(t, env.App)
	access := apptest.Connect(t, env.App, cookie, clientID, "coding:read coding:exec")
	listed := apptest.Do(t, env.App, http.MethodPost, "/api/coding",
		apptest.RPC("tools/list", "{}"), apptest.MCPHeaders(access))
	apptest.MustContain(t, listed.Body, "marimo_execute", "coding:exec")

	args := `{"url":"https://example.invalid/","token":"SYNTHETIC-CODE-123","code":"1+1"}`
	// Tool toggles and OAuth scopes still apply, independent of role.
	off := apptest.CallTool(t, env.App, access, "marimo_execute", args)
	apptest.MustContain(t, off.Body, "turned off")
	apptest.EnableTool(t, env.App, cookie, "run_command")
	missing := apptest.CallTool(t, env.App, access, "marimo_execute", args)
	apptest.MustContain(t, missing.Body, "begin_work", `"isError":true`)

	begin := apptest.CallTool(t, env.App, access, "begin_work",
		`{"role":"researcher","summary":"Check MoLab role enforcement"}`)
	var result struct {
		Result struct {
			Structured struct {
				WorkID string `json:"work_id"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(begin.Body, &result); err != nil || result.Result.Structured.WorkID == "" {
		t.Fatalf("begin_work failed: %v %s", err, begin.Body)
	}
	workID := result.Result.Structured.WorkID
	withWork := `{"work_id":"` + workID + `","url":"https://example.invalid/","token":"SYNTHETIC-CODE-123","code":"1+1"}`
	wrongRole := apptest.CallTool(t, env.App, access, "marimo_execute", withWork)
	apptest.MustContain(t, wrongRole.Body, "MoLab requires the Developer role", `"isError":true`)
	apptest.MustNotContain(t, wrongRole.Body, "SYNTHETIC-CODE-123")

	switchRole := apptest.CallTool(t, env.App, access, "set_role",
		`{"work_id":"`+workID+`","role":"developer"}`)
	apptest.MustContain(t, switchRole.Body, `"changed":true`)

	// Execution reaches the MoLab URL validator only after choosing Developer.
	out := apptest.CallTool(t, env.App, access, "marimo_execute", withWork)
	apptest.MustContain(t, out.Body, "Invalid MoLab URL")
	apptest.MustNotContain(t, out.Body, "SYNTHETIC-CODE-123")
	second := apptest.CallTool(t, env.App, access, "marimo_execute", withWork)
	apptest.MustContain(t, second.Body, "Invalid MoLab URL")

	activity := apptest.Do(t, env.App, http.MethodGet, "/api/activity", "",
		map[string]string{"Cookie": cookie})
	if strings.Contains(string(activity.Body), "SYNTHETIC-CODE-123") {
		t.Fatal("MoLab credential leaked into the activity log")
	}
	tasks := apptest.Do(t, env.App, http.MethodGet, "/api/tasks", "", apptest.JSONCookie(cookie))
	var visible []json.RawMessage
	if err := json.Unmarshal(tasks.Body, &visible); err != nil || len(visible) != 1 {
		t.Fatalf("MoLab runs must reuse the existing task: %v %s", err, tasks.Body)
	}
}

func TestChatGPTMoLabRequiresExecScope(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, clientID := apptest.SignedInClient(t, env.App)
	access := apptest.Connect(t, env.App, cookie, clientID, "coding:read")
	apptest.EnableTool(t, env.App, cookie, "run_command")
	out := apptest.CallTool(t, env.App, access, "marimo_execute",
		`{"url":"https://sb-test.sb.molab.run/","token":"SYNTHETIC-CODE-123","code":"1+1"}`)
	apptest.MustContain(t, out.Body, "insufficient_scope", "coding:exec")
}
