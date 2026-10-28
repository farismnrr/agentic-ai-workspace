package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Commit applies a compare-and-swap revision with its operation, task patch, and events.
func (s *Store) Commit(ctx context.Context, change application.WorkChange) ([]domain.Event, int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("begin work commit: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := s.now()
	revision, taskID, err := prepareWorkCommit(ctx, tx, change, now)
	if err != nil {
		return nil, 0, mapBusy(err)
	}
	events, err := applyWorkCommit(ctx, tx, s.ids, taskID, change, now)
	if err != nil {
		return nil, 0, mapBusy(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, mapBusy(fmt.Errorf("commit work changes: %w", err))
	}
	return events, revision, nil
}

func prepareWorkCommit(ctx context.Context, tx *sql.Tx, change application.WorkChange,
	now time.Time,
) (int64, string, error) {
	revision, err := advanceWork(ctx, tx, change, now)
	if err != nil {
		return 0, "", err
	}
	if change.Patch != nil && change.Patch.SupersedePlan {
		if err = supersedeSubmittedPlan(ctx, tx, change.WorkID); err != nil {
			return 0, "", err
		}
	}
	taskID, err := ownedTaskID(ctx, tx, change.WorkID, change.UserID)
	return revision, taskID, err
}

func applyWorkCommit(ctx context.Context, tx *sql.Tx, ids application.IDs, taskID string,
	change application.WorkChange, now time.Time,
) ([]domain.Event, error) {
	if change.Op != nil {
		if err := insertWorkOp(ctx, tx, *change.Op); err != nil {
			return nil, err
		}
	}
	if err := patchChangedTask(ctx, tx, taskID, change.Task, now); err != nil {
		return nil, err
	}
	return insertWorkEvents(ctx, tx, ids, taskID, change.Events, closesWork(change))
}

func insertWorkEvents(ctx context.Context, tx *sql.Tx, ids application.IDs, taskID string,
	events []domain.Event, terminal bool,
) ([]domain.Event, error) {
	if !terminal {
		if err := checkWorkEventLimit(ctx, tx, taskID, len(events)); err != nil {
			return nil, err
		}
	}
	storedEvents := make([]domain.Event, 0, len(events))
	for _, event := range events {
		stored, err := insertEvent(ctx, tx, ids, taskID, event)
		if err != nil {
			return nil, err
		}
		storedEvents = append(storedEvents, stored)
	}
	return storedEvents, nil
}

func checkWorkEventLimit(ctx context.Context, tx *sql.Tx, taskID string, additional int) error {
	var count int
	// Each external work owns one task, so its monotonic sequence is the event count.
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq), 0) FROM task_events WHERE task_id = ?`, taskID).Scan(&count)
	if err != nil {
		return fmt.Errorf("count work events: %w", err)
	}
	if count+additional > domain.MaxWorkEvents {
		return domain.ErrTooLarge
	}
	return nil
}

func patchChangedTask(ctx context.Context, tx *sql.Tx, taskID string, patch *application.TaskPatch,
	now time.Time,
) error {
	if patch == nil {
		return nil
	}
	return patchTask(ctx, tx, taskID, patch, now)
}

func value[T any](pointer *T) any {
	if pointer == nil {
		return nil
	}
	return *pointer
}

func ownedTaskID(ctx context.Context, tx *sql.Tx, workID, userID string) (string, error) {
	var taskID string
	if err := tx.QueryRowContext(ctx, `SELECT task_id FROM external_works WHERE id = ? AND user_id = ?`,
		workID, userID).Scan(&taskID); err != nil {
		return "", fmt.Errorf("find work task: %w", err)
	}
	return taskID, nil
}

func insertWorkOp(ctx context.Context, tx *sql.Tx, op application.WorkOp) error {
	result, err := tx.ExecContext(ctx, `INSERT INTO external_work_ops(work_id, op_key, tool, req_hash, response, at)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(work_id, op_key) DO NOTHING`,
		op.WorkID, op.OpKey, op.Tool, op.ReqHash, op.Response, op.At)
	if err != nil {
		return fmt.Errorf("insert work operation: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count work operation: %w", err)
	}
	if changed == 0 {
		return domain.ErrKeyReused
	}
	return nil
}

func patchTask(ctx context.Context, tx *sql.Tx, taskID string, patch *application.TaskPatch, now time.Time) error {
	fields := application.TaskPatch{}
	if patch != nil {
		fields = *patch
	}
	plan, proof, err := encodeTaskPatch(fields)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE tasks SET updated_at = ?, status = COALESCE(?, status),
		plan = COALESCE(?, plan), prd = COALESCE(?, prd), research = COALESCE(?, research),
		summary = COALESCE(?, summary), proof = COALESCE(?, proof) WHERE id = ?`,
		now.Unix(), value(fields.Status), plan, value(fields.PRD), value(fields.Research),
		value(fields.Summary), proof, taskID)
	if err != nil {
		return fmt.Errorf("update work task: %w", err)
	}
	return nil
}

func encodeTaskPatch(patch application.TaskPatch) (any, any, error) {
	var plan, proof any
	if patch.Plan != nil {
		encoded, err := json.Marshal(*patch.Plan)
		if err != nil {
			return nil, nil, fmt.Errorf("encode work plan: %w", err)
		}
		plan = string(encoded)
	}
	if patch.Proof != nil {
		encoded, err := json.Marshal(patch.Proof)
		if err != nil {
			return nil, nil, fmt.Errorf("encode work proof: %w", err)
		}
		proof = string(encoded)
	}
	return plan, proof, nil
}
