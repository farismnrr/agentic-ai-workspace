package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// DeleteProject stops anything still running in a project and removes its tasks.
func (s *Service) DeleteProject(ctx context.Context, userID, projectID string) error {
	tasks, err := s.deps.Tasks.All(ctx, userID, projectID, unlimited)
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	s.stopAndForget(ctx, tasks)

	if err := s.deps.Tasks.DeleteProject(ctx, userID, projectID); err != nil {
		return fmt.Errorf("delete tasks: %w", err)
	}

	return nil
}

// stopAndForget cancels the tasks that are running and removes their screenshots.
func (s *Service) stopAndForget(ctx context.Context, tasks []domain.Task) {
	for _, task := range tasks {
		if task.Status.Running() {
			s.dispatcher.Cancel(task.ID)
		}

		if s.deps.Shots != nil {
			_ = s.deps.Shots.Remove(ctx, task.ID)
		}
	}
}

// DeleteAll stops anything still running and removes all of a person's tasks.
func (s *Service) DeleteAll(ctx context.Context, userID string) error {
	tasks, err := s.deps.Tasks.All(ctx, userID, "", unlimited)
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	projects := map[string]struct{}{}
	for _, task := range tasks {
		if task.Status.Running() {
			s.stopTask(ctx, task)
		}
		s.removeTaskFiles(ctx, task)
		projects[task.ProjectID] = struct{}{}
	}

	if err := s.deps.Tasks.DeleteAll(ctx, userID); err != nil {
		return fmt.Errorf("delete tasks: %w", err)
	}
	for projectID := range projects {
		s.rebuildNotes(ctx, userID, projectID)
	}

	return nil
}

// Everything lists all of a person's tasks with no cap, for exporting.
func (s *Service) Everything(ctx context.Context, userID string) ([]domain.Task, error) {
	tasks, err := s.deps.Tasks.All(ctx, userID, "", unlimited)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, nil
}

// unlimited makes SQLite return every row.
const unlimited = -1
