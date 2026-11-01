package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// EnableExternalDeepReview is reachable only through the authenticated owner
// route. It cannot be requested by an external work call. Change and audit
// record share a transaction, and an already-enabled mode remains sticky.
func (s *Store) EnableExternalDeepReview(
	ctx context.Context, userID, taskID string, at time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return mapBusy(fmt.Errorf("begin external review promotion: %w", err))
	}
	defer func() { _ = tx.Rollback() }()

	const activeExternal = `source <> '' AND work_id <> '' AND archived_at IS NULL
        AND status IN ('working', 'review', 'needs-input')`
	result, err := tx.ExecContext(ctx, `UPDATE tasks
        SET review_mode = 'deep-review', updated_at = ?
        WHERE id = ? AND user_id = ? AND review_mode = 'swift' AND `+activeExternal,
		at.Unix(), taskID, userID)
	if err != nil {
		return mapBusy(fmt.Errorf("enable external deep review: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count external deep-review promotion: %w", err)
	}
	if changed == 0 {
		var saved string
		err := tx.QueryRowContext(ctx, `SELECT review_mode FROM tasks
            WHERE id = ? AND user_id = ? AND `+activeExternal, taskID, userID).Scan(&saved)
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrWrongState
		}
		if err != nil {
			return mapBusy(fmt.Errorf("read external review mode: %w", err))
		}
		if saved != string(domain.ReviewModeDeepReview) {
			return domain.ErrWrongState
		}
		return tx.Commit()
	}
	// Invalidate any completion or role mutation that read a Swift snapshot
	// before this owner action. Both rows change within one transaction.
	workResult, err := tx.ExecContext(ctx, `UPDATE external_works
		SET rev = rev + 1, updated_at = ?
		WHERE id = (SELECT work_id FROM tasks WHERE id = ? AND user_id = ?)
		AND user_id = ? AND state IN ('open', 'blocked')`,
		at.Unix(), taskID, userID, userID)
	if err != nil {
		return mapBusy(fmt.Errorf("advance external review revision: %w", err))
	}
	advanced, err := workResult.RowsAffected()
	if err != nil || advanced != 1 {
		return domain.ErrWrongState
	}
	_, err = insertEvent(ctx, tx, s.ids, taskID, domain.Event{
		Kind: domain.EventMessage, Text: "Deep Review enabled by task owner.",
		Time: at,
	})
	if err != nil {
		return mapBusy(fmt.Errorf("audit external review opt-in: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return mapBusy(fmt.Errorf("commit external review opt-in: %w", err))
	}
	return nil
}
