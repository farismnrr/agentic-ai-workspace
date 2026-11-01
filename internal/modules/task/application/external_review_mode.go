package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// ExternalDeepReviewStore is kept separate so ordinary task stores and tests
// do not gain a writable review-mode capability through connected tools.
type ExternalDeepReviewStore interface {
	EnableExternalDeepReview(ctx context.Context, userID, taskID string, at time.Time) error
}

// EnableExternalDeepReview is called only from the authenticated owner UI
// route, never from a connected assistant or an agent tool.
func (s *Service) EnableExternalDeepReview(
	ctx context.Context, userID, taskID string,
) (domain.Task, error) {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if task.Source == "" || task.WorkID == "" || task.ArchivedAt != nil {
		return domain.Task{}, domain.ErrWrongState
	}
	if task.ReviewMode == domain.ReviewModeDeepReview {
		return s.Get(ctx, userID, taskID)
	}
	if task.ReviewMode != domain.ReviewModeSwift {
		return domain.Task{}, domain.ErrBadMode
	}
	store, ok := s.deps.Tasks.(ExternalDeepReviewStore)
	if !ok {
		return domain.Task{}, domain.ErrWrongState
	}
	if err := store.EnableExternalDeepReview(ctx, userID, taskID, s.now()); err != nil {
		return domain.Task{}, err
	}
	return s.Get(ctx, userID, taskID)
}
