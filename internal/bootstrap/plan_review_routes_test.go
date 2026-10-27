package bootstrap

import (
	"encoding/json"
	"testing"
)

func submittedExternalPlan(t *testing.T) (testEnv, string, string, string, string) {
	t.Helper()
	env := newEnv(t)
	cookie, clientID := signedInClient(t, env.app)
	token := connect(t, env.app, cookie, clientID, "coding:read")
	begin := callTool(t, env.app, token, "begin_work",
		`{"summary":"Prepare a small change","role":"developer","idempotency_key":"route-work-key"}`)
	content := toolContent(t, begin)
	workID, _ := content["work_id"].(string)
	taskID, _ := content["task_id"].(string)
	if workID == "" || taskID == "" {
		t.Fatalf("begin_work omitted ids: %s", begin.Body)
	}
	callTool(t, env.app, token, "update_plan", `{"work_id":"`+workID+
		`","base_revision":0,"steps":[{"title":"Make the change","state":"pending"}]}`)
	callTool(t, env.app, token, "save_requirements", `{"work_id":"`+workID+`","markdown":"A small change."}`)
	submitted := callTool(t, env.app, token, "submit_plan", `{"work_id":"`+workID+
		`","base_revision":1,"summary":"Make one small change","small_task":true}`)
	mustContain(t, submitted.Body, "ready_for_approval")
	return env, cookie, clientID, taskID, workID
}

func planSession(cookie string) map[string]string {
	headers := jsonHeaders()
	headers["Cookie"] = cookie
	return headers
}

func toolContent(t *testing.T, response reply) map[string]any {
	t.Helper()
	var envelope struct {
		Result struct {
			Structured map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body, &envelope); err != nil {
		t.Fatalf("decode MCP result: %v: %s", err, response.Body)
	}
	return envelope.Result.Structured
}

func TestAppPlanApprovalIsOwnerBoundAndIdempotent(t *testing.T) {
	t.Parallel()
	env, cookie, clientID, taskID, workID := submittedExternalPlan(t)
	path := "/api/tasks/" + taskID

	snapshot := do(t, env.app, "GET", path+"/plan/revisions/1", "", planSession(cookie))
	if snapshot.Status != 200 {
		t.Fatalf("load submitted snapshot: %d %s", snapshot.Status, snapshot.Body)
	}
	mustContain(t, snapshot.Body, `"revision":1`, "Make one small change", "Make the change")

	other := cookieOf(t, do(t, env.app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve-review@example.com","password":"correct horse"}`, jsonHeaders()))
	wrongOwner := do(t, env.app, "POST", path+"/plan/approve",
		`{"revision":1,"idempotencyKey":"approval-key-123"}`, planSession(other))
	if wrongOwner.Status != 404 {
		t.Fatalf("wrong owner approval status=%d body=%s", wrongOwner.Status, wrongOwner.Body)
	}
	stale := do(t, env.app, "POST", path+"/plan/approve",
		`{"revision":2,"idempotencyKey":"approval-key-123"}`, planSession(cookie))
	if stale.Status != 409 {
		t.Fatalf("stale plan approval status=%d body=%s", stale.Status, stale.Body)
	}
	body := `{"revision":1,"idempotencyKey":"approval-key-123"}`
	first := do(t, env.app, "POST", path+"/plan/approve", body, planSession(cookie))
	second := do(t, env.app, "POST", path+"/plan/approve", body, planSession(cookie))
	if first.Status != 200 || second.Status != 200 {
		t.Fatalf("approval/retry statuses=%d/%d bodies=%s/%s", first.Status, second.Status, first.Body, second.Body)
	}
	var firstResult, secondResult map[string]any
	if err := json.Unmarshal(first.Body, &firstResult); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second.Body, &secondResult); err != nil {
		t.Fatal(err)
	}
	if firstResult["decision"] != "approved" || firstResult["source"] != "app" || firstResult["decidedAt"] == "" ||
		firstResult["handoffId"] == "" || firstResult["handoffId"] != secondResult["handoffId"] ||
		firstResult["linkedTaskId"] == "" || firstResult["linkedTaskId"] != secondResult["linkedTaskId"] {
		t.Fatalf("approval retry changed handoff reference: %v then %v", firstResult, secondResult)
	}
	toolHeaders := jsonCookie(cookie)
	do(t, env.app, "PUT", "/api/spaces/coding/tools/approve_plan", `{"enabled":true}`, toolHeaders)
	token := connect(t, env.app, cookie, clientID, "coding:read coding:approve")
	tool := callTool(t, env.app, token, "approve_plan", `{"work_id":"`+workID+
		`","revision":1,"idempotency_key":"mcp-retry-key"}`)
	toolResult := toolContent(t, tool)
	if toolResult["handoff_id"] != firstResult["handoffId"] {
		t.Fatalf("app and MCP paths returned different handoffs: app=%v MCP=%v response=%s",
			firstResult["handoffId"], toolResult, tool.Body)
	}
}

func TestAppCanRequestChangesForSubmittedRevision(t *testing.T) {
	t.Parallel()
	env, cookie, _, taskID, _ := submittedExternalPlan(t)
	result := do(t, env.app, "POST", "/api/tasks/"+taskID+"/plan/request-changes",
		`{"revision":1,"note":"Please clarify the first step."}`, planSession(cookie))
	if result.Status != 200 {
		t.Fatalf("request changes status=%d body=%s", result.Status, result.Body)
	}
	mustContain(t, result.Body, `"decision":"changes_requested"`)
}
