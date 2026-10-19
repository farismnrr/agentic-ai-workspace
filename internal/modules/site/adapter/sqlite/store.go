// Package sqlite stores sites.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// Store keeps sites in SQLite.
type Store struct{ db *sql.DB }

// NewStore wraps an open database.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

const (
	columns     = "project_id, user_id, slug, token, visibility, status, port, container_id, started_at, command"
	selectSites = "SELECT " + columns + " FROM sites"

	byProject = selectSites + " WHERE project_id = ?"
	bySlug    = selectSites + " WHERE slug = ?"
	byToken   = selectSites + " WHERE token = ?"
	byUser    = selectSites + " WHERE user_id = ? ORDER BY started_at DESC"
	running   = selectSites + " WHERE status = 'running' ORDER BY started_at DESC"
)

// Save inserts or replaces the project's site.
func (s *Store) Save(ctx context.Context, site domain.Site) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sites (`+columns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (project_id) DO UPDATE SET user_id = excluded.user_id, slug = excluded.slug,
		token = excluded.token, visibility = excluded.visibility, status = excluded.status,
		port = excluded.port, container_id = excluded.container_id, started_at = excluded.started_at,
		command = excluded.command`,
		site.ProjectID, site.UserID, site.Slug, site.Token, site.Visibility, site.Status, site.Port,
		site.ContainerID, site.StartedAt.Unix(), site.Command)
	if err != nil {
		return fmt.Errorf("save site: %w", err)
	}

	return nil
}

// ByProject returns a project's site.
func (s *Store) ByProject(ctx context.Context, projectID string) (domain.Site, error) {
	return s.one(ctx, byProject, projectID)
}

// BySlug returns the site an address name belongs to.
func (s *Store) BySlug(ctx context.Context, slug string) (domain.Site, error) {
	return s.one(ctx, bySlug, slug)
}

// ByToken returns the site an address token belongs to.
func (s *Store) ByToken(ctx context.Context, token string) (domain.Site, error) {
	return s.one(ctx, byToken, token)
}

func (s *Store) one(ctx context.Context, query string, arg any) (domain.Site, error) {
	site, err := scan(s.db.QueryRowContext(ctx, query, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Site{}, domain.ErrNotFound
	}

	if err != nil {
		return domain.Site{}, fmt.Errorf("select site: %w", err)
	}

	return site, nil
}

// ListByUser returns a person's sites, newest first.
func (s *Store) ListByUser(ctx context.Context, userID string) ([]domain.Site, error) {
	return s.many(ctx, byUser, userID)
}

// Running returns every running site.
func (s *Store) Running(ctx context.Context) ([]domain.Site, error) {
	return s.many(ctx, running)
}

func (s *Store) many(ctx context.Context, query string, args ...any) ([]domain.Site, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select sites: %w", err)
	}

	defer func() { _ = rows.Close() }()

	sites := []domain.Site{}

	for rows.Next() {
		site, scanErr := scan(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("read site: %w", scanErr)
		}

		sites = append(sites, site)
	}

	return sites, rows.Err() //nolint:wrapcheck // a failed iteration is reported by the caller
}

// SetVisibility changes who may open the site.
func (s *Store) SetVisibility(ctx context.Context, projectID string, visibility domain.Visibility) error {
	if _, err := s.db.ExecContext(ctx, `UPDATE sites SET visibility = ? WHERE project_id = ?`,
		visibility, projectID); err != nil {
		return fmt.Errorf("update visibility: %w", err)
	}

	return nil
}

// MarkStopped marks one site stopped, or all of them when projectID is empty.
func (s *Store) MarkStopped(ctx context.Context, projectID string) error {
	query, args := `UPDATE sites SET status = 'stopped', container_id = ''`, []any{}
	if projectID != "" {
		query, args = query+` WHERE project_id = ?`, []any{projectID}
	}

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("mark stopped: %w", err)
	}

	return nil
}

type scanner interface{ Scan(dest ...any) error }

func scan(row scanner) (domain.Site, error) {
	var (
		site    domain.Site
		started int64
	)

	err := row.Scan(&site.ProjectID, &site.UserID, &site.Slug, &site.Token, &site.Visibility, &site.Status,
		&site.Port, &site.ContainerID, &started, &site.Command)
	site.StartedAt = time.Unix(started, 0)

	return site, err //nolint:wrapcheck // callers add the context
}
