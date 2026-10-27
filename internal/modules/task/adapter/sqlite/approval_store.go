package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// DecidePlan claims the submitted revision and persists its decision/outbox atomically.
func (s *Store) DecidePlan(ctx context.Context, in application.PlanDecisionWrite) (
	application.PlanDecisionResult, error,
) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return application.PlanDecisionResult{}, mapBusy(fmt.Errorf("begin plan decision: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	result, replay, err := s.decidePlanInTransaction(ctx, tx, in)
	if err != nil {
		return application.PlanDecisionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		label := "plan decision"
		if replay {
			label = "plan decision replay"
		}
		return application.PlanDecisionResult{}, mapBusy(fmt.Errorf("commit %s: %w", label, err))
	}
	return result, nil
}

func (s *Store) decidePlanInTransaction(ctx context.Context, tx *sql.Tx, in application.PlanDecisionWrite) (
	application.PlanDecisionResult, bool, error,
) {
	revisionID, eventID, taskID, err := claimRevision(ctx, tx, in)
	if errors.Is(err, sql.ErrNoRows) {
		result, replayErr := persistedDecision(ctx, tx, in)
		return result, true, replayErr
	}
	if err != nil {
		return application.PlanDecisionResult{}, false, mapBusy(fmt.Errorf("claim plan revision: %w", err))
	}
	if taskID != in.Decision.TaskID {
		return application.PlanDecisionResult{}, false, domain.ErrUnknownWork
	}
	in.Decision.RevisionID = revisionID
	if err := persistDecision(ctx, tx, s.ids, in, eventID); err != nil {
		return application.PlanDecisionResult{}, false, mapBusy(err)
	}
	return decisionResult(in.Decision, in.HandoffID, "pending", ""), false, nil
}

func persistDecision(ctx context.Context, tx *sql.Tx, ids application.IDs, in application.PlanDecisionWrite,
	eventID string,
) error {
	if err := advanceDecisionWork(ctx, tx, in); err != nil {
		return err
	}
	if err := insertDecision(ctx, tx, in.Decision); err != nil {
		return err
	}
	if in.Decision.Kind == domain.PlanDecisionApproved {
		if err := insertPendingHandoff(ctx, tx, in); err != nil {
			return err
		}
	}
	if err := patchReviewedTask(ctx, tx, in); err != nil {
		return err
	}
	if err := updateReviewEvent(ctx, tx, in, eventID); err != nil {
		return err
	}
	return insertDecisionMessage(ctx, tx, ids, in)
}

func claimRevision(ctx context.Context, tx *sql.Tx, in application.PlanDecisionWrite) (
	revisionID, eventID, taskID string, err error,
) {
	status := string(in.Decision.Kind)
	err = tx.QueryRowContext(ctx, `UPDATE external_plan_revisions SET status = ?
		WHERE work_id = ? AND rev_no = ? AND status = 'submitted' RETURNING id, event_id, task_id`,
		status, in.Decision.WorkID, in.Revision).Scan(&revisionID, &eventID, &taskID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("claim submitted plan revision: %w", err)
	}
	return revisionID, eventID, taskID, err
}

func advanceDecisionWork(ctx context.Context, tx *sql.Tx, in application.PlanDecisionWrite) error {
	phase, state, ended := "draft", "open", any(nil)
	if in.Decision.Kind == domain.PlanDecisionApproved {
		phase, state, ended = "approved", "completed", in.Decision.DecidedAt.Unix()
	}
	result, err := tx.ExecContext(ctx, `UPDATE external_works SET rev = rev + 1, plan_state = ?, state = ?,
		ended_at = COALESCE(?, ended_at), updated_at = ? WHERE id = ? AND task_id = ? AND user_id = ?
		AND project_id = ? AND state = 'open' AND plan_state = 'ready_for_approval'
		AND (? != 'mcp' OR client = ?) AND EXISTS (SELECT 1 FROM tasks WHERE id = ? AND user_id = ?
		AND project_id = ? AND work_id = ? AND archived_at IS NULL)`,
		phase, state, ended, in.Decision.DecidedAt.Unix(), in.Decision.WorkID, in.Decision.TaskID,
		in.Decision.UserID, in.Decision.ProjectID, in.Decision.Source, in.Decision.Client,
		in.Decision.TaskID, in.Decision.UserID, in.Decision.ProjectID, in.Decision.WorkID)
	if err != nil {
		return fmt.Errorf("advance decided work: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count decided work update: %w", err)
	}
	if changed == 0 {
		return decisionWorkError(ctx, tx, in)
	}
	return nil
}

func decisionWorkError(ctx context.Context, tx *sql.Tx, in application.PlanDecisionWrite) error {
	if err := checkDecisionOwner(ctx, tx, in); err != nil {
		return err
	}
	return domain.ErrWrongState
}

func decisionResult(decision domain.PlanDecision, handoffID, state, taskID string) application.PlanDecisionResult {
	return application.PlanDecisionResult{
		Decision: decision, HandoffID: handoffID, HandoffState: state, InternalTaskID: taskID,
	}
}

var _ application.PlanDecisionStore = (*Store)(nil)
