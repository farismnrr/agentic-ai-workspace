package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// withWindow fills in the model's context window for a task that has not recorded one yet,
// so the page can show how full it is before the next model call.
func (s *Service) withWindow(ctx context.Context, task domain.Task) domain.Task {
	if task.ContextWindow == 0 && task.Model != "" && s.deps.Windows != nil {
		task.ContextWindow = s.deps.Windows.ContextWindow(ctx, task.UserID, task.ProviderID, task.Model)
	}
	if work := s.workSummary(ctx, task); work != nil {
		task.Work = work
	}
	return task
}

func (s *Service) workSummary(ctx context.Context, task domain.Task) *domain.WorkSummary {
	if task.WorkID == "" || s.workSummaries == nil {
		return nil
	}
	work, err := s.workSummaries.ByTask(ctx, task.ID)
	if err != nil {
		return nil
	}
	currentRevision, decision, handoffState, linkedTaskID := s.currentWorkReview(ctx, work)
	return &domain.WorkSummary{
		ID: work.ID, UpdatedAt: work.UpdatedAt, State: work.State, PlanState: work.PlanState, Role: work.RoleKey,
		ReportedAt: work.ReportedAt,
		RoleName:   work.RoleName, Actor: work.Client, Revision: work.Rev,
		CurrentRevision: currentRevision, PlanRev: work.PlanRev,
		Decision: decision, HandoffState: handoffState, LinkedTaskID: linkedTaskID,
		Stale: domain.DeriveWorkPhase(work, s.now(), domain.WorkPhaseThresholds{
			IdleAfter: s.workIdleAfter,
		}) == domain.WorkPhaseIdle,
	}
}

func (s *Service) currentWorkReview(ctx context.Context, work domain.ExternalWork) (int64, string, string, string) {
	if s.planSnapshots == nil {
		return work.PlanRev, "", "", ""
	}
	review, found, err := s.planSnapshots.LatestPlanReview(ctx, work.ID)
	if err != nil || !found {
		return work.PlanRev, "", "", ""
	}
	return review.Revision, review.Decision, review.HandoffState, review.LinkedTaskID
}
