package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestOrchestratorMessageCanFollowUpDoneTask(t *testing.T) {
	t.Parallel()
	r := newTeamRig(nil, []application.AgentRole{developer()})
	task := r.startPlanned(t, "Build the menu")
	ctx := context.Background()
	if err := r.mem.SetStatus(ctx, task.ID, domain.StatusDone, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := r.service.SendOrchestratorMessage(ctx, "u1", "proj", task.ID, "One more change"); err != nil {
		t.Fatalf("completed task follow-up was rejected: %v", err)
	}
	if got := r.task(t, task.ID).Status; got != domain.StatusQueued {
		t.Fatalf("follow-up did not queue the same task: %s", got)
	}
}
