// Package sqlite keeps egress grants in the egress_grants table.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

const columns = `id, project_id, task_id, host, reason, status, requested_by_kind, requested_by,
	decided_by, created_at, decided_at, expires_at`

// Store is the egress_grants table.
type Store struct{ db *sql.DB }

// NewStore builds the store over db.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Create inserts the grant in one statement that also checks, in the same atomic step, that the task belongs to the
// project and that the task has no live request for the host. Two callers racing for the same host cannot both win.
func (s *Store) Create(ctx context.Context, g domain.Grant) error {
	result, err := s.db.ExecContext(ctx, `INSERT INTO egress_grants (`+columns+`)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, '', ?, 0, ?
		WHERE EXISTS (SELECT 1 FROM tasks WHERE id = ? AND project_id = ?)
		AND NOT EXISTS (SELECT 1 FROM egress_grants WHERE task_id = ? AND host = ?
			AND status IN ('pending', 'approved') AND expires_at > ?)`,
		g.ID, g.ProjectID, g.TaskID, g.Host, g.Reason, g.Status, g.RequestedBy.Kind, g.RequestedBy.ID,
		g.CreatedAt.Unix(), g.ExpiresAt.Unix(),
		g.TaskID, g.ProjectID, g.TaskID, g.Host, g.CreatedAt.Unix())
	if err != nil {
		return fmt.Errorf("insert grant: %w", err)
	}

	created, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("insert grant: %w", err)
	}

	if created == 1 {
		return nil
	}

	return s.whyNotCreated(ctx, g)
}

func (s *Store) whyNotCreated(ctx context.Context, g domain.Grant) error {
	var owned int

	query := `SELECT COUNT(*) FROM tasks WHERE id = ? AND project_id = ?`
	if err := s.db.QueryRowContext(ctx, query, g.TaskID, g.ProjectID).Scan(&owned); err != nil {
		return fmt.Errorf("check task: %w", err)
	}

	if owned == 0 {
		return application.ErrScope
	}

	return application.ErrDuplicate
}

// Get returns one grant.
func (s *Store) Get(ctx context.Context, id string) (domain.Grant, error) {
	return scan(s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM egress_grants WHERE id = ?`, id))
}

// Active lists approved grants of the task that end after now. The task must belong to the project, so a row
// written with a mismatched pair never counts.
func (s *Store) Active(ctx context.Context, projectID, taskID string, now time.Time) ([]domain.Grant, error) {
	return s.list(ctx, `SELECT `+prefixed("g")+` FROM egress_grants g
		JOIN tasks t ON t.id = g.task_id AND t.project_id = g.project_id
		WHERE g.project_id = ? AND g.task_id = ? AND g.status = 'approved' AND g.expires_at > ?
		ORDER BY g.host`, projectID, taskID, now.Unix())
}

// List returns every grant of the task, newest first.
func (s *Store) List(ctx context.Context, projectID, taskID string) ([]domain.Grant, error) {
	return s.list(ctx, `SELECT `+columns+` FROM egress_grants WHERE project_id = ? AND task_id = ?
		ORDER BY created_at DESC, id`, projectID, taskID)
}

func (s *Store) list(ctx context.Context, query string, args ...any) ([]domain.Grant, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query grants: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var grants []domain.Grant

	for rows.Next() {
		g, scanErr := scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}

		grants = append(grants, g)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read grants: %w", err)
	}

	return grants, nil
}

type scanner interface{ Scan(dest ...any) error }

func scan(row scanner) (domain.Grant, error) {
	var (
		g                           domain.Grant
		status, kind                string
		created, decided, expiresAt int64
	)

	err := row.Scan(&g.ID, &g.ProjectID, &g.TaskID, &g.Host, &g.Reason, &status, &kind, &g.RequestedBy.ID,
		&g.DecidedBy, &created, &decided, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Grant{}, application.ErrNotFound
	}

	if err != nil {
		return domain.Grant{}, fmt.Errorf("read grant: %w", err)
	}

	g.Status, g.RequestedBy.Kind = domain.Status(status), domain.ActorKind(kind)
	g.CreatedAt, g.ExpiresAt = time.Unix(created, 0).UTC(), time.Unix(expiresAt, 0).UTC()

	if decided != 0 {
		g.DecidedAt = time.Unix(decided, 0).UTC()
	}

	return g, nil
}

func prefixed(alias string) string {
	return alias + ".id, " + alias + ".project_id, " + alias + ".task_id, " + alias + ".host, " + alias +
		".reason, " + alias + ".status, " + alias + ".requested_by_kind, " + alias + ".requested_by, " + alias +
		".decided_by, " + alias + ".created_at, " + alias + ".decided_at, " + alias + ".expires_at"
}
