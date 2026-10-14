package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// DeleteAll stops anything still running and removes all of a person's tasks.
func (s *Service) DeleteAll(ctx context.Context, userID string) error {
	tasks, err := s.deps.Tasks.List(ctx, userID, "", listLimit)
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	for _, task := range tasks {
		if task.Status.Running() {
			s.dispatcher.Cancel(task.ID)
		}
	}

	if err := s.deps.Tasks.DeleteAll(ctx, userID); err != nil {
		return fmt.Errorf("delete tasks: %w", err)
	}

	return nil
}

// Everything lists all of a person's tasks with no cap, for exporting.
func (s *Service) Everything(ctx context.Context, userID string) ([]domain.Task, error) {
	tasks, err := s.deps.Tasks.List(ctx, userID, "", unlimited)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, nil
}

// unlimited makes SQLite return every row.
const unlimited = -1
