package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// WorkStopper stops the external session linked to a task.
type WorkStopper interface {
	Stop(ctx context.Context, userID, client, projectID, workID string) error
}

// WorkConfirmer marks a reported external session complete for its owner.
type WorkConfirmer interface {
	Confirm(ctx context.Context, userID, client, projectID, workID string) (Completion, error)
}

// WorkSummaries loads safe work state for task responses.
type WorkSummaries interface {
	ByTask(ctx context.Context, taskID string) (domain.ExternalWork, error)
}

// UseWorkServices wires optional external-work behavior at the composition root.
func (s *Service) UseWorkServices(stopper WorkStopper, confirmer WorkConfirmer, summaries WorkSummaries) {
	s.workStopper, s.workConfirmer, s.workSummaries = stopper, confirmer, summaries
}

// UseExternalQuestionStore wires the transactional answer path for external work.
func (s *Service) UseExternalQuestionStore(store QuestionAnswerStore) { s.externalQuestions = store }
