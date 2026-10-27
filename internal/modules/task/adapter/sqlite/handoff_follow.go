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

func (s *Store) FollowHandoff(ctx context.Context, handoff application.PlanHandoff, task domain.Task,
	event domain.Event, prompt string, at time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return mapBusy(fmt.Errorf("begin linked plan follow-up: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	status, err := handoffTaskStatus(ctx, tx, handoff, task.ID)
	if err != nil {
		return err
	}
	exists, err := handoffEventExists(ctx, tx, task.ID, handoff.ID)
	if err != nil {
		return err
	}
	if !exists {
		if err := appendHandoffFollowUp(ctx, tx, s.ids, task.ID, status, event, prompt, at); err != nil {
			return err
		}
	}
	if err := markHandoffCreated(ctx, tx, handoff, task.ID, at); err != nil {
		return mapBusy(err)
	}
	if err := tx.Commit(); err != nil {
		return mapBusy(fmt.Errorf("commit linked plan follow-up: %w", err))
	}
	return nil
}

func appendHandoffFollowUp(ctx context.Context, tx *sql.Tx, ids application.IDs, taskID string,
	status domain.Status, event domain.Event, prompt string, at time.Time,
) error {
	if status.Running() {
		return domain.ErrBusy
	}
	if status == domain.StatusNeedsInput {
		return domain.ErrNeedsAnswer
	}
	if _, err := insertEvent(ctx, tx, ids, taskID, event); err != nil {
		return mapBusy(err)
	}
	block := domain.Block{Kind: domain.BlockText, Text: prompt}
	if err := appendTranscript(ctx, tx, taskID, domain.RoleUser, block, ""); err != nil {
		return mapBusy(fmt.Errorf("persist approved plan prompt: %w", err))
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET status = 'queued', updated_at = ? WHERE id = ?`,
		at.Unix(), taskID); err != nil {
		return mapBusy(fmt.Errorf("queue linked plan task: %w", err))
	}
	return nil
}

func handoffTaskStatus(ctx context.Context, tx *sql.Tx, handoff application.PlanHandoff, taskID string) (
	domain.Status, error,
) {
	var status string
	var archived sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT status, archived_at FROM tasks WHERE id = ? AND user_id = ?
		AND project_id = ? AND origin_task_id = ? AND source = '' AND work_id = ''`,
		taskID, handoff.UserID, handoff.ProjectID, handoff.TaskID).Scan(&status, &archived)
	if errors.Is(err, sql.ErrNoRows) || archived.Valid {
		return "", domain.ErrWrongState
	}
	if err != nil {
		return "", mapBusy(fmt.Errorf("read linked plan task: %w", err))
	}
	return domain.Status(status), nil
}

func handoffEventExists(ctx context.Context, tx *sql.Tx, taskID, handoffID string) (bool, error) {
	var found int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM task_events WHERE task_id = ? AND json_extract(body,
		'$.technical') = ? LIMIT 1`, taskID, "handoff:"+handoffID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, mapBusy(err)
}

func markHandoffCreated(ctx context.Context, tx *sql.Tx, handoff application.PlanHandoff, taskID string,
	at time.Time,
) error {
	result, err := tx.ExecContext(ctx, `UPDATE external_plan_handoffs SET state = 'created',
		internal_task_id = ?, next_attempt_at = ?, last_error = '' WHERE id = ? AND work_id = ?
		AND state IN ('pending', 'created')`, taskID, at.Add(time.Minute).Unix(), handoff.ID, handoff.WorkID)
	if err != nil {
		return fmt.Errorf("link created plan handoff: %w", err)
	}
	if err := requireOneRow(result, domain.ErrWrongState, "link created plan handoff"); err != nil {
		return err
	}
	if err := markWorkHandedOff(ctx, tx, handoff, at); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET status = 'done', updated_at = ? WHERE id = ?
		AND user_id = ? AND project_id = ? AND work_id = ?`, at.Unix(), handoff.TaskID,
		handoff.UserID, handoff.ProjectID, handoff.WorkID); err != nil {
		return fmt.Errorf("mark external work task complete: %w", err)
	}
	handoff.InternalTaskID = taskID
	return updateHandedOffCard(ctx, tx, handoff)
}

func markWorkHandedOff(ctx context.Context, tx *sql.Tx, handoff application.PlanHandoff, at time.Time) error {
	result, err := tx.ExecContext(ctx, `UPDATE external_works SET rev = rev + 1, plan_state = 'handed_off',
		updated_at = ? WHERE id = ? AND state = 'completed' AND plan_state = 'approved'`,
		at.Unix(), handoff.WorkID)
	if err != nil {
		return fmt.Errorf("mark plan phase handed off: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count handed-off work update: %w", err)
	}
	if changed == 1 {
		return nil
	}
	var phase string
	err = tx.QueryRowContext(ctx, `SELECT plan_state FROM external_works WHERE id = ?`, handoff.WorkID).Scan(&phase)
	if err != nil {
		return fmt.Errorf("verify handed-off work phase: %w", err)
	}
	if phase != string(domain.PlanHandedOff) {
		return domain.ErrWrongState
	}
	return nil
}

func updateHandedOffCard(ctx context.Context, tx *sql.Tx, handoff application.PlanHandoff) error {
	event, err := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+
		` FROM task_events WHERE task_id = ? AND id = ?`, handoff.TaskID, handoff.Revision.EventID))
	if err != nil {
		return err
	}
	if event.PlanReview == nil || event.PlanReview.Revision != handoff.Revision.RevNo {
		return domain.ErrStaleRevision
	}
	event.PlanReview.HandoffID = handoff.ID
	event.PlanReview.HandoffState = "handed_off"
	event.PlanReview.TaskID = handoff.InternalTaskID
	event.PlanReview.LinkedTaskID = handoff.InternalTaskID
	return updateEventIn(ctx, tx, handoff.TaskID, event)
}
