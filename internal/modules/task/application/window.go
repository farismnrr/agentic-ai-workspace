package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// withWindow fills in the model's context window for a task that has not recorded one yet,
// so the page can show how full it is before the next model call.
func (s *Service) withWindow(ctx context.Context, task domain.Task) domain.Task {
	if task.ContextWindow == 0 && task.Model != "" && s.deps.Windows != nil {
		task.ContextWindow = s.deps.Windows.ContextWindow(ctx, task.UserID, task.ProviderID, task.Model)
	}
	if task.WorkID != "" && s.workSummaries != nil {
		work, err := s.workSummaries.ByTask(ctx, task.ID)
		if err == nil {
			task.Work = &domain.WorkSummary{
				ID: work.ID, UpdatedAt: work.UpdatedAt, State: work.State, Role: work.RoleKey,
				RoleName: work.RoleName, Actor: work.Client, Revision: work.Rev,
				PlanRev: work.PlanRev,
				Stale:   work.State == domain.WorkOpen && time.Since(work.UpdatedAt) > workIdleAfter,
			}
		}
	}

	return task
}
