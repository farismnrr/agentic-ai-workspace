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

	return task
}
