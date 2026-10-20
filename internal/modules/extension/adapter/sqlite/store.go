// Package sqlite keeps the extensions the AI built.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/domain"
)

// Store keeps one row per project's extension.
type Store struct{ db *sql.DB }

// NewStore opens the store on the shared database.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

const columns = `project_id, user_id, slug, name, version, description, permissions, hosts, size, files, deployed_at`

// Save inserts or replaces the project's extension.
func (s *Store) Save(ctx context.Context, e domain.Extension) error {
	permissions, hosts := encode(e.Permissions), encode(e.Hosts)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO extensions (`+columns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (project_id) DO UPDATE SET user_id = excluded.user_id, slug = excluded.slug, name = excluded.name,
		   version = excluded.version, description = excluded.description, permissions = excluded.permissions,
		   hosts = excluded.hosts, size = excluded.size, files = excluded.files, deployed_at = excluded.deployed_at`,
		e.ProjectID, e.UserID, e.Slug, e.Name, e.Version, e.Description, permissions, hosts, e.Size, e.Files,
		e.DeployedAt.Unix())
	if err != nil {
		return fmt.Errorf("save extension: %w", err)
	}

	return nil
}

// ByProject returns domain.ErrNotFound when the project has no extension of this person.
func (s *Store) ByProject(ctx context.Context, userID, projectID string) (domain.Extension, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM extensions WHERE user_id = ? AND project_id = ?`,
		userID, projectID)

	extension, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Extension{}, domain.ErrNotFound
	}

	return extension, err
}

// List returns the person's extensions, newest first.
func (s *Store) List(ctx context.Context, userID string) ([]domain.Extension, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+columns+` FROM extensions WHERE user_id = ? ORDER BY deployed_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("select extensions: %w", err)
	}

	defer func() { _ = rows.Close() }()

	list := []domain.Extension{}

	for rows.Next() {
		extension, scanErr := scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}

		list = append(list, extension)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read extensions: %w", err)
	}

	return list, nil
}

// Delete removes one of the person's extensions; a missing one is domain.ErrNotFound.
func (s *Store) Delete(ctx context.Context, userID, projectID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM extensions WHERE user_id = ? AND project_id = ?`, userID, projectID)
	if err != nil {
		return fmt.Errorf("delete extension: %w", err)
	}

	if changed, countErr := result.RowsAffected(); countErr == nil && changed == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// DeleteProject removes a project's extension and returns its owner.
func (s *Store) DeleteProject(ctx context.Context, projectID string) (string, error) {
	var userID string

	err := s.db.QueryRowContext(ctx, `DELETE FROM extensions WHERE project_id = ? RETURNING user_id`,
		projectID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("delete project extension: %w", err)
	}

	return userID, nil
}

type scanner interface{ Scan(dest ...any) error }

func scan(row scanner) (domain.Extension, error) {
	var (
		e                  domain.Extension
		permissions, hosts string
		deployed           int64
	)

	err := row.Scan(&e.ProjectID, &e.UserID, &e.Slug, &e.Name, &e.Version, &e.Description, &permissions, &hosts,
		&e.Size, &e.Files, &deployed)
	if err != nil {
		return domain.Extension{}, fmt.Errorf("scan extension: %w", err)
	}

	e.Permissions, e.Hosts, e.DeployedAt = decode(permissions), decode(hosts), time.Unix(deployed, 0)

	return e, nil
}

func encode(list []string) string {
	text, err := json.Marshal(append([]string{}, list...))
	if err != nil {
		return "[]"
	}

	return string(text)
}

func decode(text string) []string {
	var list []string
	if json.Unmarshal([]byte(text), &list) != nil {
		return []string{}
	}

	return list
}
