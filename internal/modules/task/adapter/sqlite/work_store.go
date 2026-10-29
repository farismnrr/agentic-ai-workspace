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
	rev, plan_rev, next_step, idem_key, idem_hash, created_at, updated_at, ended_at, plan_state, reported_at, completion_reason`

// CreateWork atomically creates the task, work row, and initial request event.
// A repeated idempotency key returns the existing row without creating a task.
func (s *Store) CreateWork(ctx context.Context, create application.WorkCreate) (domain.ExternalWork, error) {
	task, work, events := create.Task, create.Work, create.Events
	if work.PlanState == "" {
		work.PlanState = domain.PlanDraft
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ExternalWork{}, fmt.Errorf("begin create work: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	task.WorkID = work.ID
	if insertErr := insertTask(ctx, tx, task); insertErr != nil {
		return domain.ExternalWork{}, insertErr
	}
	inserted, err := insertExternalWork(ctx, tx, work)
	if err != nil {
		return domain.ExternalWork{}, err
	}
	if !inserted {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return domain.ExternalWork{}, fmt.Errorf("rollback duplicate work: %w", rollbackErr)
		}
		return s.findIdempotent(ctx, work)
	}
	if err := insertInitialWorkEvents(ctx, tx, s.ids, task.ID, events); err != nil {
		return domain.ExternalWork{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.ExternalWork{}, fmt.Errorf("commit create work: %w", err)
	}
	return work, nil
}

func insertExternalWork(ctx context.Context, tx *sql.Tx, work domain.ExternalWork) (bool, error) {
	result, err := tx.ExecContext(ctx, `INSERT INTO external_works (`+workColumns+`) VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, client, project_id, idem_key) WHERE idem_key <> '' DO NOTHING`,
		work.ID, work.TaskID, work.UserID, work.ProjectID, work.Client, string(work.State), work.RoleKey, work.RoleName,
		work.Rev, work.PlanRev, work.NextStep, work.IdemKey, work.IdemHash, work.CreatedAt.Unix(), work.UpdatedAt.Unix(),
		nullUnix(work.EndedAt), string(work.PlanState), nullTime(work.ReportedAt), work.CompletionReason)
	if err != nil {
		return false, fmt.Errorf("insert work: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count inserted work: %w", err)
	}
	return inserted != 0, nil
}

func insertInitialWorkEvents(ctx context.Context, tx *sql.Tx, ids application.IDs, taskID string,
	events []domain.Event,
) error {
	for _, event := range events {
		if _, err := insertEvent(ctx, tx, ids, taskID, event); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) findIdempotent(ctx context.Context, work domain.ExternalWork) (domain.ExternalWork, error) {
	return scanWork(s.db.QueryRowContext(ctx, `SELECT `+workColumns+` FROM external_works
		WHERE user_id = ? AND client = ? AND project_id = ? AND idem_key = ? AND idem_key <> ''`,
		work.UserID, work.Client, work.ProjectID, work.IdemKey))
}

// FindIdempotent returns the prior work for a begin request key.
func (s *Store) FindIdempotent(
	ctx context.Context, userID, client, projectID, key string,
) (domain.ExternalWork, bool, error) {
	work, err := scanWork(s.db.QueryRowContext(ctx, `SELECT `+workColumns+` FROM external_works
		WHERE user_id = ? AND client = ? AND project_id = ? AND idem_key = ? AND idem_key <> ''`,
		userID, client, projectID, key))
	if errors.Is(err, sql.ErrNoRows) {
		var workID, hash string
		err = s.db.QueryRowContext(ctx, `SELECT work_id, idem_hash FROM external_work_begin_aliases
			WHERE user_id = ? AND client = ? AND project_id = ? AND idem_key = ?`,
			userID, client, projectID, key).Scan(&workID, &hash)
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ExternalWork{}, false, nil
		}
		if err != nil {
			return domain.ExternalWork{}, false, fmt.Errorf("find begin alias: %w", err)
		}
		work, err = scanWork(s.db.QueryRowContext(ctx, `SELECT `+workColumns+` FROM external_works
			WHERE id = ? AND user_id = ? AND client = ? AND project_id = ?`, workID, userID, client, projectID))
		if err == nil {
			work.IdemHash = hash
		}
		return work, err == nil, err
	}
	return work, err == nil, err
}

// RegisterBeginAlias records the idempotency key of a role-specific begin that
// reused an existing task. The original key's ownership is never modified.
func (s *Store) RegisterBeginAlias(ctx context.Context, userID, client, projectID, key, hash, workID string) error {
	if _, err := s.db.ExecContext(ctx, `INSERT INTO external_work_begin_aliases
		(user_id, client, project_id, idem_key, idem_hash, work_id)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING`,
		userID, client, projectID, key, hash, workID); err != nil {
		return fmt.Errorf("save begin alias: %w", err)
	}
	mapped, found, err := s.FindIdempotent(ctx, userID, client, projectID, key)
	if err != nil {
		return fmt.Errorf("verify begin alias: %w", err)
	}
	if !found || mapped.ID != workID || mapped.IdemHash != hash {
		return domain.ErrKeyReused
	}
	return nil
}

// LatestActiveWork keeps one in-progress issue per owner, connected app and project
// unless the caller explicitly starts another issue. Never joins another client.
func (s *Store) LatestActiveWork(
	ctx context.Context, userID, client, projectID string,
) (domain.ExternalWork, bool, error) {
	work, err := scanWork(s.db.QueryRowContext(ctx, `SELECT `+workColumns+` FROM external_works
        WHERE user_id = ? AND client = ? AND project_id = ?
        AND state IN ('open', 'blocked')
        AND EXISTS (SELECT 1 FROM tasks t WHERE t.id = external_works.task_id AND t.archived_at IS NULL)
        ORDER BY updated_at DESC, created_at DESC, rowid DESC LIMIT 1`,
		userID, client, projectID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ExternalWork{}, false, nil
	}
	return work, err == nil, err
}

// FindWork returns a work row only to its owner.
func (s *Store) FindWork(ctx context.Context, userID, client, projectID, workID string) (domain.ExternalWork, error) {
	work, err := scanWork(s.db.QueryRowContext(ctx,
		`SELECT `+workColumns+` FROM external_works
		 WHERE id = ? AND user_id = ? AND client = ? AND project_id = ?`, workID, userID, client, projectID))
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
