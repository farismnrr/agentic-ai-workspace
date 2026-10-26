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

// liveTask is the SQL for "the task is still open": not archived, not done, not failed. A deleted task has no row.
const liveTask = `t.archived_at IS NULL AND t.status NOT IN ('done', 'failed')`

// Store is the egress_grants table.
type Store struct{ db *sql.DB }

// NewStore builds the store over db.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Create inserts the grant in one statement that also checks, in the same atomic step, that the task is an open
// task of the project, that it has no live request for the host, and that the limits hold. Two callers racing for
// the same host or for the last free place cannot both win.
func (s *Store) Create(ctx context.Context, g domain.Grant, limits application.Limits) error {
	since := g.CreatedAt.Add(-limits.Window).Unix()

	result, err := s.db.ExecContext(ctx, `INSERT INTO egress_grants (`+columns+`)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, '', ?, 0, ?
		WHERE EXISTS (SELECT 1 FROM tasks t WHERE t.id = ? AND t.project_id = ? AND `+liveTask+`)
		AND NOT EXISTS (SELECT 1 FROM egress_grants WHERE task_id = ? AND host = ?
			AND status IN ('pending', 'approved') AND expires_at > ?)
		AND (SELECT COUNT(*) FROM egress_grants WHERE task_id = ? AND status = 'pending' AND expires_at > ?) < ?
		AND (? != 'agent' OR (SELECT COUNT(*) FROM egress_grants WHERE task_id = ?
			AND requested_by_kind = 'agent' AND created_at > ?) < ?)`,
		g.ID, g.ProjectID, g.TaskID, g.Host, g.Reason, g.Status, g.RequestedBy.Kind, g.RequestedBy.ID,
		g.CreatedAt.Unix(), g.ExpiresAt.Unix(),
		g.TaskID, g.ProjectID,
		g.TaskID, g.Host, g.CreatedAt.Unix(),
		g.TaskID, g.CreatedAt.Unix(), limits.MaxPending,
		g.RequestedBy.Kind, g.TaskID, since, limits.MaxByAgent)
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

	return s.whyNotCreated(ctx, g, limits)
}

// whyNotCreated names the first rule that stopped the insert, in the order a caller can act on them.
func (s *Store) whyNotCreated(ctx context.Context, g domain.Grant, limits application.Limits) error {
	live, err := s.TaskLive(ctx, g.ProjectID, g.TaskID)
	if err != nil {
		return err
	}

	if !live {
		return application.ErrScope
	}

	if _, err = s.Live(ctx, g.TaskID, g.Host, g.CreatedAt); err == nil {
		return application.ErrDuplicate
	}

	var pending int

	query := `SELECT COUNT(*) FROM egress_grants WHERE task_id = ? AND status = 'pending' AND expires_at > ?`
	if err = s.db.QueryRowContext(ctx, query, g.TaskID, g.CreatedAt.Unix()).Scan(&pending); err != nil {
		return fmt.Errorf("count requests: %w", err)
	}

	if pending >= limits.MaxPending {
		return application.ErrTooManyPending
	}

	return application.ErrRateLimited
}

// Get returns one grant.
func (s *Store) Get(ctx context.Context, id string) (domain.Grant, error) {
	return scan(s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM egress_grants WHERE id = ?`, id))
}

// Active lists approved grants of the task that end after now. The task must belong to the project, so a row
// written with a mismatched pair never counts.
func (s *Store) Active(ctx context.Context, projectID, taskID string, now time.Time) ([]domain.Grant, error) {
	return s.list(ctx, `SELECT `+prefixed("g")+` FROM egress_grants g
		JOIN tasks t ON t.id = g.task_id AND t.project_id = g.project_id AND `+liveTask+`
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
