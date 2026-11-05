package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (s *Store) StopExternalLeadTask(
	ctx context.Context, userID, taskID, source string, at time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return mapBusy(fmt.Errorf("begin external task stop: %w", err))
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `UPDATE tasks SET status = 'done', updated_at = ?
		WHERE id = ? AND user_id = ? AND source = ? AND status <> 'done'`,
		at.Unix(), taskID, userID, source)
	if err != nil {
		return mapBusy(fmt.Errorf("stop external task: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count stopped tasks: %w", err)
	}
	alreadyDone := changed == 0
	if alreadyDone {
		var status, currentSource string
		if err := tx.QueryRowContext(ctx, `SELECT status, source FROM tasks WHERE id = ? AND user_id = ?`,
			taskID, userID).Scan(&status, &currentSource); err != nil {
			return domain.ErrWrongState
		}
		if status != string(domain.StatusDone) || currentSource != source {
			return domain.ErrWrongState
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE task_delegations SET status = 'cancelled', ended_at = ?
		WHERE task_id = ? AND status IN ('working', 'waiting')`, at.Unix(), taskID); err != nil {
		return mapBusy(fmt.Errorf("cancel external task jobs: %w", err))
	}
	if _, err := tx.ExecContext(ctx, `UPDATE task_events
		SET body = json_set(body, '$.answered', 'Stopped by the external lead'),
		    version = (SELECT COALESCE(MAX(version), 0) + 1 FROM task_events WHERE task_id = ?)
		WHERE task_id = ? AND kind = 'question' AND json_extract(body, '$.agent') = ?
		  AND COALESCE(json_extract(body, '$.answered'), '') = ''`, taskID, taskID, source); err != nil {
		return mapBusy(fmt.Errorf("close external task approvals: %w", err))
	}
	if !alreadyDone {
		if _, err := insertEvent(ctx, tx, s.ids, taskID, domain.Event{
			Kind: domain.EventMessage, Text: "Stopped by the external lead", Time: at,
		}); err != nil {
			return mapBusy(fmt.Errorf("record external task stop: %w", err))
		}
	}
	if err := tx.Commit(); err != nil {
		return mapBusy(fmt.Errorf("commit external task stop: %w", err))
	}
	return nil
}
