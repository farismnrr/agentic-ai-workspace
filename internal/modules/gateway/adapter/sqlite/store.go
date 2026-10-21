// Package sqlite keeps each project's chosen gateway.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/gateway/application"
)

// Store is the project_gateway table.
type Store struct{ db *sql.DB }

// NewStore builds the store over db.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Get returns the provider id the project chose, or application.ErrUnset.
func (s *Store) Get(ctx context.Context, projectID string) (string, error) {
	var id string

	err := s.db.QueryRowContext(ctx, `SELECT provider_id FROM project_gateway WHERE project_id = ?`, projectID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", application.ErrUnset
	}

	if err != nil {
		return "", fmt.Errorf("load project gateway: %w", err)
	}

	return id, nil
}

// Set saves the project's choice.
func (s *Store) Set(ctx context.Context, projectID, providerID string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO project_gateway (project_id, provider_id, updated_at)
		VALUES (?, ?, ?) ON CONFLICT(project_id) DO UPDATE SET provider_id = excluded.provider_id,
		updated_at = excluded.updated_at`, projectID, providerID, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("save project gateway: %w", err)
	}

	return nil
}

// Delete forgets the project's choice.
func (s *Store) Delete(ctx context.Context, projectID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM project_gateway WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("delete project gateway: %w", err)
	}

	return nil
}
