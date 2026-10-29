package sqlite

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// SetContext records how full the model's context window is. It leaves updated_at alone,
// because a task should not jump to the top of the list just for being measured.
func (s *Store) SetContext(ctx context.Context, id string, used, window int64) error {
	return s.exec(ctx, `UPDATE tasks SET context_used = ?, context_window = ?,
	 context_state = CASE WHEN ? > 0 AND ? > 0 THEN 'measured' ELSE 'unknown' END
	 WHERE id = ?`, used, window, used, window, id)
}

// SetModel changes the model for the next run and resets the previous model's meter.
func (s *Store) SetModel(ctx context.Context, id, providerID, model string, effort domain.Effort) error {
	return s.exec(ctx, `UPDATE tasks SET provider_id = ?, model = ?, effort = ?, context_used = 0,
	 context_window = 0, context_state = 'unknown' WHERE id = ?`, providerID, model, string(effort), id)
}

// SetContextState records an in-progress or failed compaction without faking a token count.
func (s *Store) SetContextState(ctx context.Context, id, state string) error {
	return s.exec(ctx, `UPDATE tasks SET context_state = ? WHERE id = ?`, state, id)
}
