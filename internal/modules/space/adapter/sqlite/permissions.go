package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store persists tool permissions and the activity log in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore wraps an open database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ToolStates returns the tools a person switched on or off themselves.
func (s *Store) ToolStates(ctx context.Context, userID, space string) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT tool, enabled FROM tool_permissions WHERE user_id = ? AND space = ?`, userID, space)
	if err != nil {
		return nil, fmt.Errorf("select tool permissions: %w", err)
	}

	defer func() { _ = rows.Close() }()

	states := map[string]bool{}

	for rows.Next() {
		var (
			tool    string
			enabled bool
		)

		if err := rows.Scan(&tool, &enabled); err != nil {
			return nil, fmt.Errorf("scan tool permission: %w", err)
		}

		states[tool] = enabled
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tool permissions: %w", err)
	}

	return states, nil
}

// SetTool records a person's choice for one tool.
func (s *Store) SetTool(ctx context.Context, userID, space, tool string, enabled bool, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tool_permissions (user_id, space, tool, enabled, updated_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (user_id, space, tool) DO UPDATE SET enabled = excluded.enabled, updated_at = excluded.updated_at`,
		userID, space, tool, enabled, at.Unix())
	if err != nil {
		return fmt.Errorf("upsert tool permission: %w", err)
	}

	return nil
}
