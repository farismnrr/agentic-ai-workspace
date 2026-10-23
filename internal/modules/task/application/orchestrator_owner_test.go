package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestOrchestratorOperationsRejectEmptyOwner(t *testing.T) {
	t.Parallel()
	r := newTeamRig(nil, []application.AgentRole{developer()})
	task := r.startPlanned(t, "Build the menu")

	_, err := r.service.GetOrchestrator(context.Background(), "", "proj", task.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty owner read task: %v", err)
	}
}
