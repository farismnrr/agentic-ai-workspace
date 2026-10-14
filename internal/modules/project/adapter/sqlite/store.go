package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

const columns = `id, user_id, name, description, created_at, archived_at`

// Store persists projects in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore wraps an open database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type scanner interface {
	Scan(dest ...any) error
}

// Create inserts a project.
func (s *Store) Create(ctx context.Context, p domain.Project) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (id, user_id, name, description, created_at) VALUES (?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.Name, p.Description, p.CreatedAt.Unix())
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}

	return nil
}

const (
	newestFirst  = ` ORDER BY created_at DESC, rowid DESC`
	selectAll    = `SELECT ` + columns + ` FROM projects WHERE user_id = ?` + newestFirst
	selectActive = `SELECT ` + columns + ` FROM projects WHERE user_id = ? AND archived_at IS NULL` + newestFirst
)

// List returns a person's active projects, newest first.
func (s *Store) List(ctx context.Context, userID string) ([]domain.Project, error) {
	return s.list(ctx, selectActive, userID)
}

// ListAll returns every project a person has, archived ones too.
func (s *Store) ListAll(ctx context.Context, userID string) ([]domain.Project, error) {
	return s.list(ctx, selectAll, userID)
}

func (s *Store) list(ctx context.Context, query, userID string) ([]domain.Project, error) {
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("select projects: %w", err)
	}

	defer func() { _ = rows.Close() }()

	projects := []domain.Project{}

	for rows.Next() {
		project, err := scan(rows)
		if err != nil {
			return nil, err
		}

		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read projects: %w", err)
	}

	return projects, nil
}

// Find loads one active project.
func (s *Store) Find(ctx context.Context, userID, id string) (domain.Project, error) {
	project, err := scan(s.db.QueryRowContext(ctx,
		`SELECT `+columns+` FROM projects WHERE user_id = ? AND id = ? AND archived_at IS NULL`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Project{}, domain.ErrNotFound
	}

	return project, err
}

// FindByID loads one active project whoever owns it.
func (s *Store) FindByID(ctx context.Context, id string) (domain.Project, error) {
	project, err := scan(s.db.QueryRowContext(ctx,
		`SELECT `+columns+` FROM projects WHERE id = ? AND archived_at IS NULL`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Project{}, domain.ErrNotFound
	}

	return project, err
}

// Update changes the editable fields.
func (s *Store) Update(ctx context.Context, userID, id, name, description string) error {
	return s.exec(ctx,
		`UPDATE projects SET name = ?, description = ? WHERE user_id = ? AND id = ? AND archived_at IS NULL`,
		name, description, userID, id)
}

// Archive hides a project.
func (s *Store) Archive(ctx context.Context, userID, id string, at time.Time) error {
	return s.exec(ctx, `UPDATE projects SET archived_at = ? WHERE user_id = ? AND id = ? AND archived_at IS NULL`,
		at.Unix(), userID, id)
}

// DeleteAll removes a person's projects and returns their ids so folders can follow.
func (s *Store) DeleteAll(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `DELETE FROM projects WHERE user_id = ? RETURNING id`, userID)
	if err != nil {
		return nil, fmt.Errorf("delete projects: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var ids []string

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan project id: %w", err)
		}

		ids = append(ids, id)
	}

	return ids, rows.Err() //nolint:wrapcheck // plain iteration error
}

func (s *Store) exec(ctx context.Context, query string, args ...any) error {
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update projects: %w", err)
	}

	if changed, err := result.RowsAffected(); err == nil && changed == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func scan(row scanner) (domain.Project, error) {
	var (
		p        domain.Project
		created  int64
		archived sql.NullInt64
	)

	if err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &created, &archived); err != nil {
		return domain.Project{}, fmt.Errorf("scan project: %w", err)
	}

	p.CreatedAt = time.Unix(created, 0)
	if archived.Valid {
		moment := time.Unix(archived.Int64, 0)
		p.ArchivedAt = &moment
	}

	return p, nil
}
