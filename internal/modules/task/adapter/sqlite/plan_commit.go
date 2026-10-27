package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func supersedeSubmittedPlan(ctx context.Context, tx *sql.Tx, workID string) error {
	var taskID, eventID string
	err := tx.QueryRowContext(ctx, `UPDATE external_plan_revisions SET status = 'superseded'
		WHERE work_id = ? AND status = 'submitted' RETURNING task_id, event_id`, workID).
		Scan(&taskID, &eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("supersede submitted plan: %w", err)
	}
	event, err := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+
		` FROM task_events WHERE task_id = ? AND id = ?`, taskID, eventID))
	if err != nil {
		return err
	}
	if event.PlanReview != nil {
		event.PlanReview.Status = domain.PlanRevisionSuperseded
	}
	return updateEventIn(ctx, tx, taskID, event)
}

// submitPlanSnapshot writes the phase change, status and immutable revision in one transaction.
func (s *Store) submitPlanSnapshot(ctx context.Context, workID, userID string, expectRev int64,
	revision domain.PlanRevision, event domain.Event,
) (domain.PlanRevision, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.PlanRevision{}, mapBusy(err)
	}
	defer func() { _ = tx.Rollback() }()
	var taskID string
	err = tx.QueryRowContext(ctx, `UPDATE external_works SET rev = rev + 1, plan_state = 'ready_for_approval',
		updated_at = unixepoch() WHERE id = ? AND user_id = ? AND rev = ? AND plan_state = 'draft'
		RETURNING task_id`, workID, userID, expectRev).Scan(&taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PlanRevision{}, domain.ErrStaleRevision
	}
	if err != nil {
		return domain.PlanRevision{}, mapBusy(fmt.Errorf("advance plan phase: %w", err))
	}
	if taskID != revision.TaskID {
		return domain.PlanRevision{}, domain.ErrUnknownWork
	}
	revision, err = s.persistPlanSnapshot(ctx, tx, workID, taskID, revision, event)
	if err != nil {
		return domain.PlanRevision{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.PlanRevision{}, mapBusy(fmt.Errorf("commit submitted plan: %w", err))
	}
	return revision, nil
}

func (s *Store) persistPlanSnapshot(ctx context.Context, tx *sql.Tx, workID, taskID string,
	revision domain.PlanRevision, event domain.Event,
) (domain.PlanRevision, error) {
	if numberErr := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(rev_no), 0) + 1 FROM external_plan_revisions
		WHERE work_id = ?`, workID).Scan(&revision.RevNo); numberErr != nil {
		return domain.PlanRevision{}, fmt.Errorf("number plan revision: %w", numberErr)
	}
	if event.PlanReview != nil {
		event.PlanReview.Revision = revision.RevNo
		event.PlanReview.SmallTask = revision.SmallTask
		event.PlanReview.Steps = revision.Steps
	}
	if _, updateErr := tx.ExecContext(ctx, `UPDATE tasks SET status = 'needs-input', updated_at = unixepoch()
		WHERE id = ?`, taskID); updateErr != nil {
		return domain.PlanRevision{}, mapBusy(fmt.Errorf("mark plan awaiting approval: %w", updateErr))
	}
	storedEvent, err := insertEvent(ctx, tx, s.ids, taskID, event)
	if err != nil {
		return domain.PlanRevision{}, mapBusy(err)
	}
	steps, err := json.Marshal(revision.Steps)
	if err != nil {
		return domain.PlanRevision{}, fmt.Errorf("encode plan revision steps: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO external_plan_revisions
		(id, work_id, task_id, rev_no, plan_rev, steps, prd, research, qa, small_task, summary, digest, event_id,
		 status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'submitted', unixepoch())`,
		revision.ID, revision.WorkID, revision.TaskID, revision.RevNo, revision.PlanRev, string(steps), revision.PRD,
		revision.Research, revision.QA, revision.SmallTask, revision.Summary, revision.Digest, storedEvent.ID)
	if err != nil {
		return domain.PlanRevision{}, mapBusy(fmt.Errorf("insert plan revision: %w", err))
	}
	revision.EventID = storedEvent.ID
	revision.Status = domain.PlanRevisionSubmitted
	return revision, nil
}

func mapBusy(err error) error {
	if err != nil && (strings.Contains(err.Error(), "SQLITE_BUSY") ||
		strings.Contains(err.Error(), "database is locked")) {
		return domain.ErrTryAgain
	}
	return err
}
