package sqlite

import (
	"context"
	"fmt"
)

// DeleteProject removes a person's tasks in one project; their records cascade.
func (s *Store) DeleteProject(ctx context.Context, userID, projectID string) error {
	const query = `DELETE FROM tasks WHERE user_id = ? AND project_id = ?`
	if _, err := s.db.ExecContext(ctx, query, userID, projectID); err != nil {
		return fmt.Errorf("delete project tasks: %w", err)
	}
	return nil
}

// DeleteAll removes a person's tasks; their records cascade.
func (s *Store) DeleteAll(ctx context.Context, userID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete tasks: %w", err)
	}
	return nil
}
