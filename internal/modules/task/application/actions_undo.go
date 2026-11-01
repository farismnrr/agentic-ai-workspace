package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// UndoChange puts back what an earlier step changed, unless it changed again since.
func (s *Service) UndoChange(ctx context.Context, userID, taskID, eventID string) error {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if task.ArchivedAt != nil {
		return domain.ErrWrongState
	}

	event, err := s.deps.Events.FindEvent(ctx, taskID, eventID)
	if err != nil {
		return fmt.Errorf("find event: %w", err)
	}

	if !event.CanUndo() {
		return domain.ErrWrongState
	}

	if task.Status.Running() && task.Source == "" {
		return domain.ErrBusy
	}

	if err := s.deps.Tools.Undo(ctx, task.ProjectID, event.Undo); err != nil {
		return fmt.Errorf("undo: %w", err)
	}

	event.Undone = true

	if err := s.deps.Events.UpdateEvent(ctx, taskID, event); err != nil {
		return fmt.Errorf("record undo: %w", err)
	}

	return nil
}
