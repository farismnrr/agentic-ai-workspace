package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const workColumns = `id, task_id, user_id, project_id, client, state, role_key, role_name,
	rev, plan_rev, next_step, idem_key, idem_hash, created_at, updated_at, ended_at`

// CreateWork creates the durable work row and links it to its existing task.
// A repeated idempotency key returns the existing row.
func (s *Store) CreateWork(ctx context.Context, work domain.ExternalWork) (domain.ExternalWork, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ExternalWork{}, fmt.Errorf("begin create work: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `INSERT INTO external_works (`+workColumns+`) VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, client, project_id, idem_key) WHERE idem_key <> '' DO NOTHING`,
		work.ID, work.TaskID, work.UserID, work.ProjectID, work.Client, string(work.State), work.RoleKey, work.RoleName,
		work.Rev, work.PlanRev, work.NextStep, work.IdemKey, work.IdemHash, work.CreatedAt.Unix(), work.UpdatedAt.Unix(),
		nullUnix(work.EndedAt))
	if err != nil {
		return domain.ExternalWork{}, fmt.Errorf("insert work: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return domain.ExternalWork{}, fmt.Errorf("count inserted work: %w", err)
	}
	if inserted == 0 {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return domain.ExternalWork{}, fmt.Errorf("rollback duplicate work: %w", rollbackErr)
		}
		return s.findIdempotent(ctx, work)
	}

	result, err = tx.ExecContext(ctx, `UPDATE tasks SET work_id = ? WHERE id = ? AND user_id = ? AND project_id = ?`,
		work.ID, work.TaskID, work.UserID, work.ProjectID)
	if err != nil {
		return domain.ExternalWork{}, fmt.Errorf("link work task: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return domain.ExternalWork{}, fmt.Errorf("count linked work task: %w", err)
	}
	if changed == 0 {
		return domain.ExternalWork{}, domain.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return domain.ExternalWork{}, fmt.Errorf("commit create work: %w", err)
	}
	return work, nil
}

func (s *Store) findIdempotent(ctx context.Context, work domain.ExternalWork) (domain.ExternalWork, error) {
	return scanWork(s.db.QueryRowContext(ctx, `SELECT `+workColumns+` FROM external_works
		WHERE user_id = ? AND client = ? AND project_id = ? AND idem_key = ? AND idem_key <> ''`,
		work.UserID, work.Client, work.ProjectID, work.IdemKey))
}

// FindWork returns a work row only to its owner.
func (s *Store) FindWork(ctx context.Context, userID, workID string) (domain.ExternalWork, error) {
	work, err := scanWork(s.db.QueryRowContext(ctx,
		`SELECT `+workColumns+` FROM external_works WHERE id = ? AND user_id = ?`, workID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ExternalWork{}, domain.ErrUnknownWork
	}
	return work, err
}

// ByTask finds the work session linked to a task.
func (s *Store) ByTask(ctx context.Context, taskID string) (domain.ExternalWork, error) {
	work, err := scanWork(s.db.QueryRowContext(ctx,
		`SELECT `+workColumns+` FROM external_works WHERE task_id = ?`, taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ExternalWork{}, domain.ErrUnknownWork
	}
	return work, err
}

func nullTime(at *time.Time) any {
	if at == nil {
		return nil
	}
	return at.Unix()
}

func nullUnix(at int64) any {
	if at == 0 {
		return nil
	}
	return at
}

var _ application.WorkStore = (*Store)(nil)
