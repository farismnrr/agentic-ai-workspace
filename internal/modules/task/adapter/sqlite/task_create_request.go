package sqlite

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// CreateWithRequest stores a new task and its first owner message together.
func (s *Store) CreateWithRequest(ctx context.Context, task domain.Task, event domain.Event, prompt string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin task creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertTask(ctx, tx, task); err != nil {
		return err
	}
	if _, err := insertEvent(ctx, tx, s.ids, task.ID, event); err != nil {
		return fmt.Errorf("record first task request: %w", err)
	}
	if err := appendTranscript(ctx, tx, task.ID, domain.RoleUser,
		domain.Block{Kind: domain.BlockText, Text: prompt}, ""); err != nil {
		return fmt.Errorf("record first task message: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit task creation: %w", err)
	}
	return nil
}
