package connectors_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestOrchestratorApprovePlanOptInAndFreshRevision(t *testing.T) {
	env := apptest.NewEnv(t)
	cookie, client := apptest.SignedInClient(t, env.App)
	token := orchestratorToken(t, env.App, cookie, client)
	withApproveScope := orchestratorToken(t, env.App, cookie, client, "orchestrator:approve")

	project := apptest.Do(t, env.App, "POST", "/api/projects",
		`{"name":"Approval boundary"}`, apptest.JSONCookie(cookie)).JSON(t)["id"].(string)
	for _, name := range []string{"create_task", "save_task_requirements", "update_task_plan", "submit_plan", "save_file"} {
		orchestratorOn(t, env.App, cookie, name)
	}

	task := orchestratorData(t, orchestratorCall(t, env.App, token, "create_task",
		`{"project":"`+project+`","summary":"Approval lifecycle check","idempotency_key":"approval-task-123456"}`))["ID"].(string)
	orchestratorData(t, orchestratorCall(t, env.App, token, "save_task_requirements",
		`{"task_id":"`+task+`","markdown":"# PRD\\nApproval is required before writing files."}`))
	orchestratorData(t, orchestratorCall(t, env.App, token, "update_task_plan",
		`{"task_id":"`+task+`","steps":[{"id":"first","title":"Implement","state":"active"}]}`))
	submitted := orchestratorData(t, orchestratorCall(t, env.App, token, "submit_plan",
		`{"task_id":"`+task+`"}`))
	eventID := submitted["event_id"].(string)
	if submitted["status"] != "pending" || eventID == "" {
		t.Fatalf("PRD did not reach approval: %+v", submitted)
	}

	fileArgs := `{"project":"` + project + `","task_id":"` + task + `","path":"approval-proof.txt","content":"allowed"}`
	deniedFile := orchestratorCall(t, env.App, token, "save_file", fileArgs)
	apptest.MustContain(t, deniedFile.Body, `"isError":true`)

	args := `{"task_id":"` + task + `","event_id":"` + eventID + `","idempotency_key":"owner-test-approval-1234"}`
	deniedOff := orchestratorCall(t, env.App, withApproveScope, "approve_plan", args)
	apptest.MustContain(t, deniedOff.Body, `"isError":true`)
	orchestratorOn(t, env.App, cookie, "approve_plan")

	noScope := orchestratorCall(t, env.App, token, "approve_plan", args)
	apptest.MustContain(t, noScope.Body, `"isError":true`, "orchestrator:approve")

	wrongEvent := orchestratorCall(t, env.App, withApproveScope, "approve_plan",
		`{"task_id":"`+task+`","event_id":"missing","idempotency_key":"owner-test-approval-1234"}`)
	apptest.MustContain(t, wrongEvent.Body, `"isError":true`)

	otherClient := apptest.Do(t, env.App, "POST", "/oauth/register",
		`{"client_name":"Claude","redirect_uris":["`+apptest.Redirect+`"],"token_endpoint_auth_method":"none"}`,
		apptest.JSONHeaders()).JSON(t)["client_id"].(string)
	otherToken := orchestratorToken(t, env.App, cookie, otherClient, "orchestrator:approve")
	wrongClient := orchestratorCall(t, env.App, otherToken, "approve_plan", args)
	apptest.MustContain(t, wrongClient.Body, `"isError":true`)

	result := orchestratorData(t, orchestratorCall(t, env.App, withApproveScope, "approve_plan", args))
	if result["status"] != "approved" || result["event_id"] != eventID {
		t.Fatalf("approval result: %+v", result)
	}
	again := orchestratorData(t, orchestratorCall(t, env.App, withApproveScope, "approve_plan", args))
	if again["status"] != "approved" {
		t.Fatalf("idempotent retry: %+v", again)
	}

	otherKey := orchestratorCall(t, env.App, withApproveScope, "approve_plan",
		`{"task_id":"`+task+`","event_id":"`+eventID+`","idempotency_key":"different-approval-1234"}`)
	apptest.MustContain(t, otherKey.Body, `"isError":true`)

	state := orchestratorData(t, orchestratorCall(t, env.App, token, "get_approval_state",
		`{"task_id":"`+task+`"}`))
	if state["status"] != "approved" {
		t.Fatalf("approval not visible: %+v", state)
	}
	orchestratorData(t, orchestratorCall(t, env.App, token, "save_file", fileArgs))
	if data, err := os.ReadFile(filepath.Join(env.DataDir, "projects", project, "approval-proof.txt")); err != nil || string(data) != "allowed" {
		t.Fatalf("approved write failed: %q %v", data, err)
	}

	// A later plan revision invalidates the old event even with the same key.
	orchestratorData(t, orchestratorCall(t, env.App, token, "update_task_plan",
		`{"task_id":"`+task+`","steps":[{"id":"second","title":"Revise the plan","state":"active"}]}`))
	stale := orchestratorCall(t, env.App, withApproveScope, "approve_plan", args)
	apptest.MustContain(t, stale.Body, `"isError":true`)
	state = orchestratorData(t, orchestratorCall(t, env.App, token, "get_approval_state",
		`{"task_id":"`+task+`"}`))
	if state["status"] == "approved" {
		t.Fatalf("stale revision approved: %+v", state)
	}
}
