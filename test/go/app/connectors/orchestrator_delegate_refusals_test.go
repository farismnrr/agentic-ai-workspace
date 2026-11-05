package connectors_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestOrchestratorDelegationRefusalsExplainNextStep(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := orchestratorToken(t, env.App, cookie, client)
	project := apptest.Do(t, env.App, "POST", "/api/projects",
		`{"name":"Delegation refusals"}`, apptest.JSONCookie(cookie)).JSON(t)["id"].(string)
	for _, name := range []string{
		"create_task", "delegate_agent", "update_task_plan", "save_task_requirements", "submit_plan", "complete_task",
	} {
		orchestratorOn(t, env.App, cookie, name)
	}

	create := func(key string) string {
		t.Helper()
		return orchestratorData(t, orchestratorCall(t, env.App, token, "create_task",
			`{"project":"`+project+`","summary":"Check delegation guidance","idempotency_key":"`+key+`"}`))["ID"].(string)
	}
	delegate := func(task, role, key string) apptest.Reply {
		t.Helper()
		return orchestratorCall(t, env.App, token, "delegate_agent",
			`{"task_id":"`+task+`","agent_key":"`+role+`","brief":"Do the approved work",`+
				`"idempotency_key":"`+key+`"}`)
	}
	assertRefusal := func(reply apptest.Reply, code, remedy string) {
		t.Helper()
		apptest.MustContain(t, reply.Body, `"isError":true`, code, remedy)
	}

	missingPlan := create("missing-plan-task-123")
	assertRefusal(delegate(missingPlan, "researcher", "missing-plan-job-123"), "PLAN_REQUIRED", "update_task_plan")

	approvalTask := create("approval-task-123456")
	orchestratorData(t, orchestratorCall(t, env.App, token, "save_task_requirements",
		`{"task_id":"`+approvalTask+`","markdown":"# PRD\nApproval is required."}`))
	orchestratorData(t, orchestratorCall(t, env.App, token, "update_task_plan",
		`{"task_id":"`+approvalTask+`","steps":[{"id":"build","title":"Build the feature","state":"active"}]}`))
	orchestratorData(t, orchestratorCall(t, env.App, token, "submit_plan", `{"task_id":"`+approvalTask+`"}`))
	assertRefusal(delegate(approvalTask, "developer", "approval-job-123456"), "APPROVAL_REQUIRED", "approval")

	finished := create("finished-task-12345")
	orchestratorData(t, orchestratorCall(t, env.App, token, "complete_task",
		`{"task_id":"`+finished+`","summary":"Finished"}`))
	assertRefusal(delegate(finished, "researcher", "finished-job-123456"), "TASK_NOT_ACTIVE", "waiting for a person or review")
	assertRefusal(delegate(create("policy-task-123456"), "unknown-role", "policy-job-123456"),
		"ROLE_NOT_ALLOWED", "list_agents")
}

func TestExternalResearcherCanRunWhilePlanApprovalIsPending(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := orchestratorToken(t, env.App, cookie, client)
	gateway := apptest.NewGateway(t,
		apptest.ToolCall("r1", "list_files", map[string]string{"path": "."}),
		apptest.ToolCall("r2", "git", map[string][]string{"args": {"status"}}),
		apptest.ToolCall("r3", "git", map[string][]string{"args": {"add", "--", "__nonexistent_preapproval_probe__"}}),
		apptest.ToolCall("r4", "report", map[string]any{
			"summary":  "Read-only research completed before implementation approval.",
			"document": "# Research\\nThe project files were inspected without changes.",
		}),
	)
	provider := apptest.AddProvider(t, env, cookie, gateway.Server.URL, "k").JSON(t)["id"].(string)
	if response := apptest.Do(t, env.App, "PUT", "/api/preferences",
		`{"defaultProviderId":"`+provider+`","defaultModel":"fake-model","rules":[]}`,
		apptest.JSONCookie(cookie)); response.Status != 200 && response.Status != 204 {
		t.Fatalf("model preference failed: %d %s", response.Status, response.Body)
	}
	project := apptest.Do(t, env.App, "POST", "/api/projects",
		`{"name":"Research during approval"}`, apptest.JSONCookie(cookie)).JSON(t)["id"].(string)
	apptest.EnableTool(t, env.App, cookie, "list_files")
	apptest.EnableTool(t, env.App, cookie, "git")
	token = orchestratorToken(t, env.App, cookie, client)
	for _, name := range []string{
		"create_task", "save_task_requirements", "update_task_plan", "submit_plan",
		"delegate_agent", "get_delegation",
	} {
		orchestratorOn(t, env.App, cookie, name)
	}
	task := orchestratorData(t, orchestratorCall(t, env.App, token, "create_task",
		`{"project":"`+project+`","summary":"Research an unapproved implementation plan",`+
			`"idempotency_key":"pending-research-task-12345"}`))["ID"].(string)
	orchestratorData(t, orchestratorCall(t, env.App, token, "save_task_requirements",
		`{"task_id":"`+task+`","markdown":"# PRD\nResearch may proceed before implementation approval."}`))
	orchestratorData(t, orchestratorCall(t, env.App, token, "update_task_plan",
		`{"task_id":"`+task+`","steps":[{"id":"build","title":"Implement after approval",`+
			`"state":"pending"}]}`))
	submitted := orchestratorData(t, orchestratorCall(t, env.App, token, "submit_plan",
		`{"task_id":"`+task+`"}`))
	eventID, _ := submitted["event_id"].(string)
	if submitted["status"] != "pending" || eventID == "" {
		t.Fatalf("plan did not stay pending approval: %+v", submitted)
	}
	brief := `{"task_id":"` + task + `","agent_key":"researcher",` +
		`"brief":"Inspect project files read-only and report research",` +
		`"file_scope":["docs"],"idempotency_key":"pending-research-job-12345"}`
	started := orchestratorData(t, orchestratorCall(t, env.App, token, "delegate_agent", brief))
	jobID, _ := started["job_id"].(string)
	if jobID == "" {
		t.Fatalf("Researcher job was not created: %+v", started)
	}
	retry := orchestratorData(t, orchestratorCall(t, env.App, token, "delegate_agent", brief))
	if retry["job_id"] != jobID {
		t.Fatalf("same-key retry created a different job: first=%+v retry=%+v", started, retry)
	}
	var completed bool
	for attempt := 0; attempt < 200; attempt++ {
		state := orchestratorData(t, orchestratorCall(t, env.App, token, "get_delegation",
			`{"task_id":"`+task+`","job_id":"`+jobID+`"}`))
		if state["status"] == "done" {
			completed = true
			break
		}
		if state["status"] == "failed" {
			t.Fatalf("Researcher job failed: %+v", state)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !completed {
		t.Fatal("Researcher job did not finish")
	}
	gateway.Mu.Lock()
	requestCount := len(gateway.Requests)
	gateway.Mu.Unlock()
	if requestCount < 4 {
		t.Fatalf("Researcher did not complete the scripted Git read and mutation attempt: %d requests", requestCount)
	}
	guardResult, err := json.Marshal(gateway.Sent(3))
	if err != nil || !strings.Contains(string(guardResult), "Git and GitHub changes require") {
		t.Fatalf("mutating Git call did not receive the pre-approval guard refusal: %s %v", guardResult, err)
	}
	assertRefusal := func(reply apptest.Reply, code string) {
		t.Helper()
		apptest.MustContain(t, reply.Body, `"isError":true`, code)
	}
	developer := orchestratorCall(t, env.App, token, "delegate_agent",
		`{"task_id":"`+task+`","agent_key":"developer","brief":"Implement before approval",`+
			`"file_scope":["docs"],"idempotency_key":"pending-developer-job-12345"}`)
	assertRefusal(developer, "APPROVAL_REQUIRED")
	foreignClient := apptest.Do(t, env.App, "POST", "/oauth/register",
		`{"client_name":"Second research client","redirect_uris":["`+apptest.Redirect+`"],"token_endpoint_auth_method":"none"}`,
		apptest.JSONCookie(cookie)).JSON(t)["client_id"].(string)
	foreignToken := orchestratorToken(t, env.App, cookie, foreignClient)
	foreign := orchestratorCall(t, env.App, foreignToken, "delegate_agent",
		`{"task_id":"`+task+`","agent_key":"researcher","brief":"Use another client",`+
			`"file_scope":["docs"],"idempotency_key":"foreign-pending-research-123"}`)
	apptest.MustContain(t, foreign.Body, `"isError":true`)
	otherCookie := apptest.Register(t, env.App, "Other owner", "pending-research-other@example.com", apptest.TestPassword)
	otherClient := apptest.Do(t, env.App, "POST", "/oauth/register",
		`{"client_name":"Other research owner","redirect_uris":["`+apptest.Redirect+`"],"token_endpoint_auth_method":"none"}`,
		apptest.JSONHeaders()).JSON(t)["client_id"].(string)
	otherToken := orchestratorToken(t, env.App, otherCookie, otherClient)
	otherOwner := orchestratorCall(t, env.App, otherToken, "delegate_agent",
		`{"task_id":"`+task+`","agent_key":"researcher","brief":"Use another owner's task",`+
			`"file_scope":["docs"],"idempotency_key":"other-owner-pending-research-123"}`)
	apptest.MustContain(t, otherOwner.Body, `"isError":true`)
	var pendingQuestions int
	if err := env.DB.QueryRow(
		`SELECT count(*) FROM task_events WHERE task_id = ? AND kind = 'question' AND `+
			`COALESCE(json_extract(body, '$.answered'), '') = ''`, task,
	).Scan(&pendingQuestions); err != nil || pendingQuestions != 1 {
		t.Fatalf("research changed pending question count: %d %v", pendingQuestions, err)
	}
	var currentEvent string
	if err := env.DB.QueryRow(
		`SELECT id FROM task_events WHERE task_id = ? AND kind = 'question' AND `+
			`COALESCE(json_extract(body, '$.answered'), '') = ''`, task,
	).Scan(&currentEvent); err != nil || currentEvent != eventID {
		t.Fatalf("pending approval event changed: got %q want %q: %v", currentEvent, eventID, err)
	}
	state := apptest.Do(t, env.App, "GET", "/api/tasks/"+task, "", apptest.JSONCookie(cookie)).JSON(t)
	if state["status"] != "needs-input" {
		t.Fatalf("Researcher changed task status: %+v", state)
	}
}
