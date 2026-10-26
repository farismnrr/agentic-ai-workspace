package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

	revision, err := advanceWork(ctx, tx, change)
	if err != nil {
		return nil, 0, err
	}
	taskID, err := ownedTaskID(ctx, tx, change.WorkID, change.UserID)
	if err != nil {
		return nil, 0, err
	}
	if change.Op != nil {
		if err := insertWorkOp(ctx, tx, *change.Op); err != nil {
			return nil, 0, err
		}
	}
	if err := patchTask(ctx, tx, taskID, change.Task); err != nil {
		return nil, 0, err
	}
	events := make([]domain.Event, 0, len(change.Events))
	for _, event := range change.Events {
		stored, err := insertEvent(ctx, tx, s.ids, taskID, event)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, stored)
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("commit work changes: %w", err)
	}
	return events, revision, nil
}

func advanceWork(ctx context.Context, tx *sql.Tx, change application.WorkChange) (int64, error) {
	var patch application.WorkPatch
	if change.Patch != nil {
		patch = *change.Patch
	}
	result, err := tx.ExecContext(ctx, `UPDATE external_works SET rev = rev + 1, updated_at = ?,
		state = COALESCE(?, state), role_key = COALESCE(?, role_key), role_name = COALESCE(?, role_name),
		plan_rev = COALESCE(?, plan_rev), next_step = COALESCE(?, next_step), ended_at = COALESCE(?, ended_at)
		WHERE id = ? AND user_id = ? AND rev = ?`, time.Now().Unix(), value(patch.State), value(patch.RoleKey),
		value(patch.RoleName), value(patch.PlanRev), value(patch.NextStep), nullTime(patch.EndedAt),
		change.WorkID, change.UserID, change.ExpectRev)
	if err != nil {
		return 0, fmt.Errorf("advance work revision: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count work revision: %w", err)
	}
	if changed == 0 {
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM external_works WHERE id = ? AND user_id = ?`,
			change.WorkID, change.UserID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, domain.ErrUnknownWork
		}
		if err != nil {
			return 0, fmt.Errorf("check work after revision conflict: %w", err)
		}
		return 0, domain.ErrStaleRevision
	}
	return change.ExpectRev + 1, nil
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

func patchTask(ctx context.Context, tx *sql.Tx, taskID string, patch *application.TaskPatch) error {
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
		time.Now().Unix(), value(fields.Status), plan, value(fields.PRD), value(fields.Research),
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
