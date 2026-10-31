package review_test

import (
	"encoding/json"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func submittedExternalPlan(t *testing.T) (apptest.Environment, string, string, string, string) {
	t.Helper()
	env := apptest.NewEnv(t)
	cookie, clientID := apptest.SignedInClient(t, env.App)
	token := apptest.Connect(t, env.App, cookie, clientID, "coding:read")
	begin := apptest.CallTool(t, env.App, token, "begin_work",
		`{"summary":"Prepare a small change","role":"planner","idempotency_key":"route-work-key"}`)
	content := toolContent(t, begin)
	workID, _ := content["work_id"].(string)
	taskID, _ := content["task_id"].(string)
	if workID == "" || taskID == "" {
		t.Fatalf("begin_work omitted ids: %s", begin.Body)
	}
	apptest.CallTool(t, env.App, token, "update_plan", `{"work_id":"`+workID+
		`","base_revision":0,"steps":[{"title":"Make the change","state":"pending"}]}`)
	apptest.CallTool(t, env.App, token, "save_requirements", `{"work_id":"`+workID+`","markdown":"A small change."}`)
	submitted := apptest.CallTool(t, env.App, token, "submit_plan", `{"work_id":"`+workID+
		`","base_revision":1,"summary":"Make one small change","small_task":true}`)
	apptest.MustContain(t, submitted.Body, "ready_for_approval")
	// Reviewer owns optional MCP approval; changing roles does not create a task.
	reviewer := apptest.CallTool(t, env.App, token, "set_role",
		`{"work_id":"`+workID+`","role":"reviewer"}`)
	apptest.MustContain(t, reviewer.Body, `"changed":true`)
	return env, cookie, clientID, taskID, workID
}

func planSession(cookie string) map[string]string {
	headers := apptest.JSONHeaders()
	headers["Cookie"] = cookie
	return headers
}

func toolContent(t *testing.T, response apptest.Reply) map[string]any {
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
	assertSubmittedPlanSnapshot(t, env, path, cookie)
	assertApprovalRejectsWrongOwnerAndStaleRevision(t, env, path, cookie)
	handoffID := approvePlanTwice(t, env, path, cookie)
	assertMCPApprovalSharesHandoff(t, env, cookie, clientID, workID, handoffID)
}

func assertSubmittedPlanSnapshot(t *testing.T, env apptest.Environment, path, cookie string) {
	t.Helper()
	snapshot := apptest.Do(t, env.App, "GET", path+"/plan/revisions/1", "", planSession(cookie))
	if snapshot.Status != 200 {
		t.Fatalf("load submitted snapshot: %d %s", snapshot.Status, snapshot.Body)
	}
	apptest.MustContain(t, snapshot.Body, `"revision":1`, "Make one small change", "Make the change")
}

func assertApprovalRejectsWrongOwnerAndStaleRevision(t *testing.T, env apptest.Environment, path, cookie string) {
	t.Helper()
	other := apptest.CookieOf(t, apptest.Do(t, env.App, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve-review@example.com","password":"correct horse"}`, apptest.JSONHeaders()))
	wrongOwner := apptest.Do(t, env.App, "POST", path+"/plan/approve",
		`{"revision":1,"idempotencyKey":"approval-key-123"}`, planSession(other))
	if wrongOwner.Status != 404 {
		t.Fatalf("wrong owner approval status=%d body=%s", wrongOwner.Status, wrongOwner.Body)
	}
	stale := apptest.Do(t, env.App, "POST", path+"/plan/approve",
		`{"revision":2,"idempotencyKey":"approval-key-123"}`, planSession(cookie))
	if stale.Status != 409 {
		t.Fatalf("stale plan approval status=%d body=%s", stale.Status, stale.Body)
	}
}

func approvePlanTwice(t *testing.T, env apptest.Environment, path, cookie string) string {
	t.Helper()
	body := `{"revision":1,"idempotencyKey":"approval-key-123"}`
	first := apptest.Do(t, env.App, "POST", path+"/plan/approve", body, planSession(cookie))
	second := apptest.Do(t, env.App, "POST", path+"/plan/approve", body, planSession(cookie))
	if first.Status != 200 || second.Status != 200 {
		t.Fatalf("approval/retry statuses=%d/%d bodies=%s/%s", first.Status, second.Status, first.Body, second.Body)
	}
	firstResult := decodePlanApproval(t, first.Body)
	secondResult := decodePlanApproval(t, second.Body)
	assertPlanApprovalReplayMatches(t, firstResult, secondResult)
	return firstResult["handoffId"].(string)
}

func decodePlanApproval(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func assertPlanApprovalReplayMatches(t *testing.T, first, second map[string]any) {
	t.Helper()
	if first["decision"] != "approved" || first["source"] != "app" || first["decidedAt"] == "" ||
		first["handoffId"] == "" || first["handoffId"] != second["handoffId"] ||
		first["linkedTaskId"] == "" || first["linkedTaskId"] != second["linkedTaskId"] {
		t.Fatalf("approval retry changed handoff reference: %v then %v", first, second)
	}
}

func assertMCPApprovalSharesHandoff(t *testing.T, env apptest.Environment, cookie, clientID, workID, handoffID string) {
	t.Helper()
	toolHeaders := apptest.JSONCookie(cookie)
	apptest.Do(t, env.App, "PUT", "/api/spaces/coding/tools/approve_plan", `{"enabled":true}`, toolHeaders)
	token := apptest.Connect(t, env.App, cookie, clientID, "coding:read coding:approve")
	// The Reviewer role was selected before the human approved the plan.
	tool := apptest.CallTool(t, env.App, token, "approve_plan", `{"work_id":"`+workID+
		`","revision":1,"idempotency_key":"mcp-retry-key"}`)
	toolResult := toolContent(t, tool)
	if toolResult["handoff_id"] != handoffID {
		t.Fatalf("app and MCP paths returned different handoffs: app=%v MCP=%v response=%s",
			handoffID, toolResult, tool.Body)
	}
}

func TestAppCanRequestChangesForSubmittedRevision(t *testing.T) {
	t.Parallel()
	env, cookie, _, taskID, _ := submittedExternalPlan(t)
	result := apptest.Do(t, env.App, "POST", "/api/tasks/"+taskID+"/plan/request-changes",
		`{"revision":1,"note":"Please clarify the first step."}`, planSession(cookie))
	if result.Status != 200 {
		t.Fatalf("request changes status=%d body=%s", result.Status, result.Body)
	}
	apptest.MustContain(t, result.Body, `"decision":"changes_requested"`)
}

func TestApprovedPlanHandoffDoesNotSpamTopLevelTasks(t *testing.T) {
	env, cookie, _, taskID, _ := submittedExternalPlan(t)
	gateway := apptest.NewGateway(t, apptest.FinishCall("Done"))
	provider := apptest.AddProvider(t, env, cookie, gateway.Server.URL, "k").JSON(t)["id"].(string)
	prefs := apptest.Do(t, env.App, "PUT", "/api/preferences", `{"defaultProviderId":"`+provider+`","defaultModel":"fake-model","rules":[]}`, planSession(cookie))
	if prefs.Status != 200 {
		t.Fatalf("provider preference: %d %s", prefs.Status, prefs.Body)
	}
	body := `{"revision":1,"idempotencyKey":"handoff-no-task-spam"}`
	approve := apptest.Do(t, env.App, "POST", "/api/tasks/"+taskID+"/plan/approve", body, planSession(cookie))
	if approve.Status != 200 {
		t.Fatalf("approval: %d %s", approve.Status, approve.Body)
	}
	approved := apptest.Do(t, env.App, "POST", "/api/tasks/"+taskID+"/plan/approve", body, planSession(cookie))
	if approved.Status != 200 {
		t.Fatalf("settled approval: %d %s", approved.Status, approved.Body)
	}
	linked, _ := decodePlanApproval(t, approved.Body)["linkedTaskId"].(string)
	if linked == "" || linked == taskID {
		t.Fatalf("handoff missing child ID: %s", approve.Body)
	}
	list := apptest.Do(t, env.App, "GET", "/api/tasks", "", planSession(cookie))
	var visible []struct {
		ID     string `json:"id"`
		Origin string `json:"originTaskId"`
	}
	if err := json.Unmarshal(list.Body, &visible); err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].ID != taskID {
		t.Fatalf("handoff leaked internal task: %+v", visible)
	}
	child := apptest.Do(t, env.App, "GET", "/api/tasks/"+linked, "", planSession(cookie))
	if child.Status != 200 || child.JSON(t)["originTaskId"] != taskID {
		t.Fatalf("child history inaccessible: %d %s", child.Status, child.Body)
	}
	retry := apptest.Do(t, env.App, "POST", "/api/tasks/"+taskID+"/plan/approve", body, planSession(cookie))
	if retry.Status != 200 || decodePlanApproval(t, retry.Body)["linkedTaskId"] != linked {
		t.Fatalf("handoff replay generated another child: %s", retry.Body)
	}
}
