package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestListOrchestratorTasksRequiresOwnedProject(t *testing.T) {
	t.Parallel()
	r := newTeamRig(nil, []application.AgentRole{developer()})
	_, err := r.service.ListOrchestratorTasks(context.Background(), "u1", "other-project")
	if !errors.Is(err, domain.ErrNoProject) {
		t.Fatalf("unknown project was not hidden: %v", err)
	}
}
