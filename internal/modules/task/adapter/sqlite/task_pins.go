package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Pin keeps the original pin time on repeated calls and hides tasks the person does not own.
func (s *Store) Pin(ctx context.Context, userID, id string, at time.Time) (domain.Task, error) {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET pinned_at = COALESCE(pinned_at, ?) WHERE id = ? AND user_id = ?`,
		at.Unix(), id, userID); err != nil {
		return domain.Task{}, fmt.Errorf("pin task: %w", err)
	}

	return s.Find(ctx, userID, id)
}

// Unpin clears a pin and hides tasks the person does not own.
func (s *Store) Unpin(ctx context.Context, userID, id string) (domain.Task, error) {
	const query = `UPDATE tasks SET pinned_at = NULL WHERE id = ? AND user_id = ?`
	if _, err := s.db.ExecContext(ctx, query, id, userID); err != nil {
		return domain.Task{}, fmt.Errorf("unpin task: %w", err)
	}

	return s.Find(ctx, userID, id)
}
