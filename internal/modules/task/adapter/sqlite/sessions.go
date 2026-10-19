package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Session returns the conversation a program keeps for a task; empty when it has none yet.
func (s *Store) Session(ctx context.Context, taskID string) (string, error) {
	var session string

	err := s.db.QueryRowContext(ctx, `SELECT session FROM task_cli_sessions WHERE task_id = ?`, taskID).Scan(&session)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("select session: %w", err)
	}

	return session, nil
}

// SaveSession remembers the conversation a program keeps for a task.
func (s *Store) SaveSession(ctx context.Context, taskID, session string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO task_cli_sessions (task_id, session) VALUES (?, ?)
		 ON CONFLICT (task_id) DO UPDATE SET session = excluded.session`, taskID, session)
	if err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return nil
}
