package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// UsePlanReviews gives MCP work calls the same approval service used by app routes.
func (w *Works) UsePlanReviews(reviews *PlanReviews) { w.planReviews = reviews }

// ApproveFromMCP resolves task/project identity from the authenticated work session.
func (w *Works) ApproveFromMCP(ctx context.Context, userID, client, projectID, workID string,
	revision int64, idempotencyKey string,
) (PlanDecisionResult, error) {
	if w.planReviews == nil {
		return PlanDecisionResult{}, domain.ErrWrongState
	}
	if userID == "" {
		return PlanDecisionResult{}, domain.ErrUnverifiedUser
	}
	work, err := w.store.FindWork(ctx, userID, client, projectID, workID)
	if err != nil {
		return PlanDecisionResult{}, domain.ErrUnknownWork
	}
	if work.RoleKey == "" {
		return PlanDecisionResult{}, domain.ErrBadRole
	}
	if _, err := w.availableRole(ctx, userID, work.RoleKey); err != nil {
		return PlanDecisionResult{}, err
	}
	return w.planReviews.ApproveFromMCP(ctx, userID, client, PlanReviewInput{
		ProjectID: projectID, TaskID: work.TaskID, WorkID: work.ID, Revision: revision,
		IdempotencyKey: idempotencyKey,
	})
}
