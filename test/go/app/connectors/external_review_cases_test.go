package connectors_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestExternalDeepReviewSensitiveCodeNeedsRecordedCriticalRun(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	apptest.EnableTool(t, env.App, cookie, "write_file")
	token := apptest.Connect(t, env.App, cookie, client, "coding:read coding:write coding:exec")
	started := mcpBegin(t, &env, token, `{"summary":"Modify app source","role":"developer"}`)
	workID, taskID := started.Result.Structured.WorkID, started.Result.Structured.TaskID
	args := func(values map[string]any) string {
		values["work_id"] = workID
		raw, _ := json.Marshal(values)
		return string(raw)
	}
	call := func(name string, values map[string]any) string {
		return string(apptest.CallTool(t, env.App, token, name, args(values)).Body)
	}
	enabled := apptest.Do(t, env.App, "POST", "/api/tasks/"+taskID+"/deep-review",
		"{}", apptest.JSONCookie(cookie))
	if enabled.Status != 200 {
		t.Fatalf("owner opt-in: %d %s", enabled.Status, enabled.Body)
	}
	write := call("write_file", map[string]any{"path": "backend.go", "content": "package main\n"})
	if strings.Contains(write, `"isError":true`) {
		t.Fatalf("write code: %s", write)
	}
	role := call("set_role", map[string]any{"role": "security"})
	if strings.Contains(role, `"isError":true`) {
		t.Fatalf("switch to Security: %s", role)
	}
	progress := call("report_progress", map[string]any{
		"message": "Verdict: PASS\nSecurity Assessment: Reviewed executable code.\nArchitecture Assessment: Reviewed code contracts.\nAssessment: backend.go code path checked.\nChecks:\n- go test ./...",
	})
	if strings.Contains(progress, `"isError":true`) {
		t.Fatalf("progress: %s", progress)
	}
	completed := call("complete_work", map[string]any{"outcome": "completed", "summary": "Reviewed"})
	if !strings.Contains(completed, `"isError":true`) {
		t.Fatalf("unrecorded critical scenario was accepted: %s", completed)
	}
	blocked := call("complete_work", map[string]any{"outcome": "blocked", "summary": "Need a recorded critical check"})
	if strings.Contains(blocked, `"isError":true`) {
		t.Fatalf("explicit blocker must remain available: %s", blocked)
	}
}

func TestExternalDeepReviewDocsOnlyMayCompleteWithoutSecurity(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	apptest.EnableTool(t, env.App, cookie, "write_file")
	token := apptest.Connect(t, env.App, cookie, client, "coding:read coding:write")
	started := mcpBegin(t, &env, token, `{"summary":"Update documentation only","role":"developer"}`)
	workID, taskID := started.Result.Structured.WorkID, started.Result.Structured.TaskID
	before := apptest.Do(t, env.App, "GET", "/api/tasks/"+taskID, "",
		apptest.JSONCookie(cookie)).JSON(t)
	beforeWork, ok := before["work"].(map[string]any)
	if !ok {
		t.Fatalf("external work summary missing: %+v", before)
	}
	beforeRev, ok := beforeWork["revision"].(float64)
	if !ok {
		t.Fatalf("external work revision missing: %+v", beforeWork)
	}
	promotion := apptest.Do(t, env.App, "POST", "/api/tasks/"+taskID+"/deep-review",
		"{}", apptest.JSONCookie(cookie))
	if promotion.Status != 200 {
		t.Fatalf("enable: %d %s", promotion.Status, promotion.Body)
	}
	after := apptest.Do(t, env.App, "GET", "/api/tasks/"+taskID, "",
		apptest.JSONCookie(cookie)).JSON(t)
	afterWork, ok := after["work"].(map[string]any)
	if !ok || afterWork["revision"].(float64) <= beforeRev {
		t.Fatalf("owner mode promotion did not invalidate the work revision: before=%v after=%v", beforeWork, afterWork)
	}
	written := apptest.CallTool(t, env.App, token, "write_file", `{"work_id":"`+workID+`","path":"README.md","content":"Updated docs"}`)
	if strings.Contains(string(written.Body), `"isError":true`) {
		t.Fatalf("docs write: %s", written.Body)
	}
	result := apptest.CallTool(t, env.App, token, "complete_work",
		`{"work_id":"`+workID+`","outcome":"completed","summary":"Documentation updated"}`)
	if strings.Contains(string(result.Body), `"isError":true`) {
		t.Fatalf("docs-only work requires unnecessary Security: %s", result.Body)
	}
}
