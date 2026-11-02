package connectors_test

import (
	"testing"

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
