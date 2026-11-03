package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// CloseStaleNeedsInput closes an internal task only when no unanswered owner input exists.
func (s *Store) CloseStaleNeedsInput(ctx context.Context, userID, taskID string, at time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin stale task closure: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM tasks WHERE id = ? AND user_id = ?
		AND source = '' AND status = 'needs-input' AND archived_at IS NULL`, taskID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return domain.ErrWrongState
	}
	if err != nil {
		return fmt.Errorf("check stale task state: %w", err)
	}

	var pending int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM task_events WHERE task_id = ? AND
		((kind = 'question' AND COALESCE(json_extract(body, '$.answered'), '') = '') OR
		 (kind = 'approval' AND COALESCE(json_extract(body, '$.resolved'), '') = '')) LIMIT 1`, taskID).Scan(&pending)
	if err == nil {
		return domain.ErrWrongState
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("check pending task input: %w", err)
	}

	changed, err := tx.ExecContext(ctx, `UPDATE tasks SET status = 'done', updated_at = ?
		WHERE id = ? AND user_id = ? AND source = '' AND status = 'needs-input'
		AND archived_at IS NULL AND NOT EXISTS (
			SELECT 1 FROM task_events WHERE task_id = tasks.id AND
			((kind = 'question' AND COALESCE(json_extract(body, '$.answered'), '') = '') OR
			 (kind = 'approval' AND COALESCE(json_extract(body, '$.resolved'), '') = ''))
		)`, at.Unix(), taskID, userID)
	if err != nil {
		return fmt.Errorf("compare-and-set stale task status: %w", err)
	}
	count, err := changed.RowsAffected()
	if err != nil {
		return fmt.Errorf("count stale task changes: %w", err)
	}
	if count != 1 {
		return domain.ErrWrongState
	}

	if _, err := insertEvent(ctx, tx, s.ids, taskID, domain.Event{
		Kind: domain.EventMessage, Text: "Closed by you as unverified", Time: at,
	}); err != nil {
		return fmt.Errorf("record stale task closure: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit stale task closure: %w", err)
	}
	return nil
}
