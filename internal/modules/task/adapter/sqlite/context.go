package sqlite

import "context"

// SetContext records how full the model's context window is. It leaves updated_at alone,
// because a task should not jump to the top of the list just for being measured.
func (s *Store) SetContext(ctx context.Context, id string, used, window int64) error {
	return s.exec(ctx, `UPDATE tasks SET context_used = ?, context_window = ? WHERE id = ?`, used, window, id)
}
