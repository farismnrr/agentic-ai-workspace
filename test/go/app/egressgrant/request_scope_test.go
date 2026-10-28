package egressgrant_test

import (
	"context"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestRequestEgressIgnoresForgedModelScope(t *testing.T) {
	argument := map[string]string{
		"host": "Marimo.Example.com", "reason": "open the notebook", "project_id": "p2",
		"task_id": "t2", "status": "approved",
	}
	rig := apptest.NewAgentRigWith(t, func(cfg *config.Config) { cfg.EgressGrants = true },
		apptest.ToolCall("e1", "request_egress", argument), apptest.FinishCall("Asked for access."))
	taskID := rig.Start(t, "Open the notebook")
	rig.WaitFor(t, taskID, "done")
	assertRequestUsesOnlyTrustedScope(t, rig, taskID)
}

func assertRequestUsesOnlyTrustedScope(t *testing.T, rig apptest.AgentRig, taskID string) {
	t.Helper()
	waitForGrantFinalization(t, rig, taskID)

	var projectID, storedTask, status, actor, decider string
	err := rig.Environment.DB.QueryRowContext(context.Background(), `
		SELECT project_id, task_id, status, requested_by_kind, decided_by
		FROM egress_grants WHERE task_id = ?`, taskID).Scan(&projectID, &storedTask, &status, &actor, &decider)
	if err != nil {
		t.Fatal(err)
	}
	if projectID != rig.Project || storedTask != taskID || status != "denied" || actor != "agent" ||
		decider != "system:task-ended" {
		t.Fatalf("forged scope changed the request: project %q task %q status %q actor %q decider %q",
			projectID, storedTask, status, actor, decider)
	}
	var forged int
	if err := rig.Environment.DB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM egress_grants WHERE task_id = 't2'`).Scan(&forged); err != nil {
		t.Fatal(err)
	}
	if forged != 0 {
		t.Fatalf("model created %d request(s) under its supplied task", forged)
	}
}

func waitForGrantFinalization(t *testing.T, rig apptest.AgentRig, taskID string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		var status string
		err := rig.Environment.DB.QueryRowContext(context.Background(),
			`SELECT status FROM egress_grants WHERE task_id = ?`, taskID).Scan(&status)
		if err != nil {
			t.Fatal(err)
		}
		if status != "pending" {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("the finished task's request stayed pending")
}
