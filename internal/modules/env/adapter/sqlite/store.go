package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/domain"
)

// Store persists sealed environment variables in SQLite.
type Store struct{ db *sql.DB }

// NewStore wraps an open database.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// List returns a project's stored variables without opening their values.
func (s *Store) List(ctx context.Context, projectID string) ([]application.StoredVariable, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, value_sealed, is_secret, updated_at FROM project_env_vars WHERE project_id = ? ORDER BY name`,
		projectID)
	if err != nil {
		return nil, fmt.Errorf("select project variables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	variables := make([]application.StoredVariable, 0)
	for rows.Next() {
		var item application.StoredVariable
		var updated int64
		if err := rows.Scan(&item.Name, &item.Sealed, &item.IsSecret, &updated); err != nil {
			return nil, fmt.Errorf("scan project variable: %w", err)
		}
		item.UpdatedAt = time.Unix(updated, 0).UTC()
		variables = append(variables, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read project variables: %w", err)
	}

	return variables, nil
}

// Put inserts a variable or atomically replaces the existing value.
func (s *Store) Put(ctx context.Context, projectID string, variable application.StoredVariable) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin project variable save: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM project_env_vars WHERE project_id = ? AND name = ?)`,
		projectID, variable.Name).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check project variable: %w", err)
	}
	if !exists {
		var count int
		if countErr := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM project_env_vars WHERE project_id = ?`, projectID).Scan(&count); countErr != nil {
			return fmt.Errorf("count project variables: %w", countErr)
		}
		if count >= domain.MaxVariables {
			return domain.ErrLimit
		}
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO project_env_vars
		(project_id, name, value_sealed, is_secret, updated_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(project_id, name) DO UPDATE SET
		value_sealed = excluded.value_sealed, is_secret = excluded.is_secret, updated_at = excluded.updated_at`,
		projectID, variable.Name, variable.Sealed, variable.IsSecret, variable.UpdatedAt.Unix())
	if err != nil {
		return fmt.Errorf("save project variable: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit project variable save: %w", err)
	}
	return nil
}

// Delete removes one variable.
func (s *Store) Delete(ctx context.Context, projectID, name string) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM project_env_vars WHERE project_id = ? AND name = ?`, projectID, name)
	if err != nil {
		return fmt.Errorf("delete project variable: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted project variable: %w", err)
	}
	if changed == 0 {
		return domain.ErrNotFound
	}

	return nil
}
