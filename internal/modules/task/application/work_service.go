package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// WorkStopper stops the external session linked to a task.
type WorkStopper interface {
	Stop(ctx context.Context, userID, client, projectID, workID string) error
}

// WorkSummaries loads safe work state for task responses.
type WorkSummaries interface {
	ByTask(ctx context.Context, taskID string) (domain.ExternalWork, error)
}

// UseWorkServices wires optional external-work behavior at the composition root.
func (s *Service) UseWorkServices(stopper WorkStopper, summaries WorkSummaries) {
	s.workStopper, s.workSummaries = stopper, summaries
}
