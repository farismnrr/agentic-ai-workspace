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

const handoffColumns = `h.id, h.decision_id, h.work_id, h.state, h.attempts,
	COALESCE(h.internal_task_id, ''), w.task_id, w.user_id, w.project_id, w.client,
	r.id, r.work_id, r.task_id, r.rev_no, r.plan_rev, r.steps, r.prd, r.research, r.qa,
	r.small_task, r.summary, r.digest, r.event_id, r.status, r.created_at`

func (s *Store) DueHandoffs(ctx context.Context, now time.Time, limit int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM external_plan_handoffs
		WHERE state IN ('pending', 'created') AND next_attempt_at <= ? ORDER BY next_attempt_at LIMIT ?`,
		now.Unix(), limit)
	if err != nil {
		return nil, fmt.Errorf("select due plan handoffs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan due plan handoff: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due plan handoffs: %w", err)
	}
	return ids, nil
}

func (s *Store) ClaimHandoff(ctx context.Context, id string, now time.Time) (application.PlanHandoff, bool, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE external_plan_handoffs SET
		attempts = attempts + CASE WHEN state = 'pending' THEN 1 ELSE 0 END, next_attempt_at = ?
		WHERE id = ? AND state IN ('pending', 'created') AND next_attempt_at <= ?`,
		now.Add(time.Minute).Unix(), id, now.Unix())
	if err != nil {
		return application.PlanHandoff{}, false, mapBusy(fmt.Errorf("claim plan handoff: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return application.PlanHandoff{}, false, fmt.Errorf("count claimed plan handoff: %w", err)
	}
	if changed == 0 {
		return application.PlanHandoff{}, false, nil
	}
	row, err := readHandoff(s.db.QueryRowContext(ctx, `SELECT `+handoffColumns+
		` FROM external_plan_handoffs h JOIN external_plan_decisions d ON d.id = h.decision_id
		JOIN external_works w ON w.id = h.work_id JOIN external_plan_revisions r ON r.id = d.revision_id
		WHERE h.id = ?`, id))
	return row, err == nil, err
}

func readHandoff(row scanner) (application.PlanHandoff, error) {
	var handoff application.PlanHandoff
	var status string
	var steps string
	var small int
	var created int64
	err := row.Scan(&handoff.ID, &handoff.DecisionID, &handoff.WorkID, &handoff.State,
		&handoff.Attempts, &handoff.InternalTaskID, &handoff.TaskID, &handoff.UserID,
		&handoff.ProjectID, &handoff.Client, &handoff.Revision.ID, &handoff.Revision.WorkID,
		&handoff.Revision.TaskID, &handoff.Revision.RevNo, &handoff.Revision.PlanRev,
		&steps, &handoff.Revision.PRD, &handoff.Revision.Research, &handoff.Revision.QA, &small,
		&handoff.Revision.Summary, &handoff.Revision.Digest, &handoff.Revision.EventID,
		&status, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return application.PlanHandoff{}, domain.ErrNotFound
		}
		return application.PlanHandoff{}, mapBusy(fmt.Errorf("scan plan handoff: %w", err))
	}
	if err := json.Unmarshal([]byte(steps), &handoff.Revision.Steps); err != nil {
		return application.PlanHandoff{}, fmt.Errorf("decode handoff plan: %w", err)
	}
	handoff.Revision.Status = domain.PlanRevisionStatus(status)
	handoff.Revision.SmallTask = small != 0
	handoff.Revision.CreatedAt = unixTime(created)
	return handoff, nil
}

func (s *Store) MarkHandoffDone(ctx context.Context, id string, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE external_plan_handoffs SET state = 'done',
		last_error = '', next_attempt_at = ? WHERE id = ? AND state = 'created'`, now.Unix(), id)
	if err != nil {
		return mapBusy(fmt.Errorf("mark plan handoff done: %w", err))
	}
	return requireOneRow(result, domain.ErrNotFound, "mark plan handoff done")
}

func (s *Store) FailHandoff(ctx context.Context, id, category string, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE external_plan_handoffs SET
		state = CASE WHEN attempts >= 8 THEN 'failed' ELSE 'pending' END,
		next_attempt_at = CASE WHEN attempts >= 8 THEN next_attempt_at ELSE ? END, last_error = ?
		WHERE id = ? AND state = 'pending'`, now.Add(handoffBackoff(ctx, s.db, id)).Unix(), category, id)
	if err != nil {
		return mapBusy(fmt.Errorf("record plan handoff failure: %w", err))
	}
	_, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count failed plan handoff update: %w", err)
	}
	return nil
}

func (s *Store) ResetHandoffLeases(ctx context.Context, now time.Time) error {
	if _, err := s.db.ExecContext(ctx, `UPDATE external_plan_handoffs SET state = 'created', next_attempt_at = ?
		WHERE state = 'done' AND internal_task_id IN (SELECT id FROM tasks WHERE status = 'queued')`,
		now.Unix()); err != nil {
		return mapBusy(err)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE external_plan_handoffs SET next_attempt_at = ?
		WHERE state IN ('pending', 'created')`, now.Unix())
	return mapBusy(err)
}

func (s *Store) RecoveringHandoff(ctx context.Context, taskID string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM external_plan_handoffs
		WHERE internal_task_id = ? AND state = 'created'`, taskID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check recovering plan handoff: %w", err)
	}
	return true, nil
}

var _ application.HandoffStore = (*Store)(nil)
