package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// InterruptRunningCalls marks orphaned calls without changing work activity time.
func (s *Store) InterruptRunningCalls(ctx context.Context, before *time.Time) ([]domain.ExternalWork, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin running call interruption: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	works, err := runningCalls(ctx, tx, before)
	if err != nil {
		return nil, err
	}
	changed := make([]domain.ExternalWork, 0, len(works))
	for _, call := range works {
		if err := interruptCall(ctx, tx, call); err != nil {
			return nil, err
		}
		changed = append(changed, call.work)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit running call interruptions: %w", err)
	}
	return changed, nil
}

type runningCall struct {
	work  domain.ExternalWork
	event domain.Event
}

func runningCalls(ctx context.Context, tx *sql.Tx, before *time.Time) ([]runningCall, error) {
	all := before == nil
	cutoff := int64(0)
	if before != nil {
		cutoff = before.Unix()
	}
	rows, err := tx.QueryContext(ctx, `SELECT w.id, w.task_id, w.user_id, e.body
		FROM external_works w JOIN task_events e ON e.task_id = w.task_id
		WHERE w.state = 'open' AND w.plan_state NOT IN ('ready_for_approval', 'awaiting_clarification')
		AND (? = 1 OR w.updated_at <= ?)
		AND json_extract(e.body, '$.work.id') = w.id
		AND json_extract(e.body, '$.work.phase') = 'running' ORDER BY w.task_id, e.seq`, all, cutoff)
	if err != nil {
		return nil, fmt.Errorf("select running work calls: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var calls []runningCall
	for rows.Next() {
		var work domain.ExternalWork
		var event domain.Event
		var body string
		if err := rows.Scan(&work.ID, &work.TaskID, &work.UserID, &body); err != nil {
			return nil, fmt.Errorf("scan running work call: %w", err)
		}
		if err := json.Unmarshal([]byte(body), &event); err != nil {
			return nil, fmt.Errorf("decode running work call: %w", err)
		}
		if event.Work == nil {
			continue
		}
		calls = append(calls, runningCall{work: work, event: event})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read running work calls: %w", err)
	}
	return calls, nil
}

func interruptCall(ctx context.Context, tx *sql.Tx, call runningCall) error {
	call.event.Work.Phase = "interrupted"
	call.event.Text = "Lost contact before this step finished"
	body, err := json.Marshal(call.event)
	if err != nil {
		return fmt.Errorf("encode interrupted work call: %w", err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE task_events SET body = ?, version =
		(SELECT COALESCE(MAX(version), 0) + 1 FROM task_events WHERE task_id = ?)
		WHERE task_id = ? AND id = ? AND json_extract(body, '$.work.phase') = 'running'`,
		string(body), call.work.TaskID, call.work.TaskID, call.event.ID)
	if err != nil {
		return fmt.Errorf("interrupt running work call: %w", err)
	}
	return nil
}
