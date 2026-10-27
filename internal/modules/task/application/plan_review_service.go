package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// UsePlanReviewServices wires the shared approval use case and durable dispatcher.
func (s *Service) UsePlanReviewServices(reviews *PlanReviews, handoffs *Handoffs, snapshots PlanSnapshotReader) {
	s.planReviews, s.handoffs, s.planSnapshots = reviews, handoffs, snapshots
}

// ApprovePlanFromApp uses the browser session identity supplied by its HTTP adapter.
func (s *Service) ApprovePlanFromApp(ctx context.Context, userID string, in PlanReviewInput) (
	PlanDecisionResult, error,
) {
	if s.planReviews == nil {
		return PlanDecisionResult{}, domain.ErrWrongState
	}
	return s.planReviews.ApproveFromApp(ctx, userID, in)
}

// RequestPlanChanges returns the exact submitted plan to its planner.
func (s *Service) RequestPlanChanges(ctx context.Context, userID string, in PlanReviewInput) (
	PlanDecisionResult, error,
) {
	if s.planReviews == nil {
		return PlanDecisionResult{}, domain.ErrWrongState
	}
	return s.planReviews.RequestChanges(ctx, userID, in)
}

// PlanReviewContext resolves a task through the session owner before a plan can be reviewed.
func (s *Service) PlanReviewContext(ctx context.Context, userID, taskID string, revision int64) (
	domain.PlanRevision, error,
) {
	task, work, err := s.reviewWork(ctx, userID, taskID)
	if err != nil {
		return domain.PlanRevision{}, err
	}
	snapshot, found, err := s.planSnapshots.PlanRevision(ctx, work.ID, revision)
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("read submitted plan revision: %w", err)
	}
	if !found || snapshot.TaskID != task.ID || snapshot.WorkID != work.ID {
		return domain.PlanRevision{}, domain.ErrNotFound
	}
	return snapshot, nil
}

func (s *Service) reviewWork(ctx context.Context, userID, taskID string) (domain.Task, domain.ExternalWork, error) {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return domain.Task{}, domain.ExternalWork{}, err
	}
	if task.WorkID == "" || s.workSummaries == nil || s.planSnapshots == nil {
		return domain.Task{}, domain.ExternalWork{}, domain.ErrUnknownWork
	}
	work, err := s.workSummaries.ByTask(ctx, task.ID)
	if err != nil || work.UserID != userID || work.ProjectID != task.ProjectID || work.ID != task.WorkID {
		return domain.Task{}, domain.ExternalWork{}, domain.ErrUnknownWork
	}
	return task, work, nil
}

// PlanReviewInput resolves the owner-bound work identifiers for an app decision.
func (s *Service) PlanReviewInput(ctx context.Context, userID, taskID string, revision int64,
	idempotencyKey, note string,
) (PlanReviewInput, error) {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return PlanReviewInput{}, err
	}
	snapshot, err := s.PlanReviewContext(ctx, userID, taskID, revision)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return PlanReviewInput{}, domain.ErrStaleRevision
		}
		return PlanReviewInput{}, err
	}
	return PlanReviewInput{
		ProjectID: task.ProjectID, TaskID: snapshot.TaskID, WorkID: snapshot.WorkID,
		Revision: revision, IdempotencyKey: idempotencyKey, Note: note,
	}, nil
}
