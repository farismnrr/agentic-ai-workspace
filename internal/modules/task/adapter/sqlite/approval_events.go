package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func insertDecision(ctx context.Context, tx *sql.Tx, decision domain.PlanDecision) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO external_plan_decisions
		(id, revision_id, work_id, task_id, user_id, project_id, decision, source, client, idem_key,
		req_hash, note, decided_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		decision.ID, decision.RevisionID, decision.WorkID, decision.TaskID, decision.UserID, decision.ProjectID,
		string(decision.Kind), decision.Source, decision.Client, decision.IdemKey, decision.ReqHash,
		decision.Note, decision.DecidedAt.Unix())
	if err != nil {
		keyErr := reusedDecisionKey(ctx, tx, decision)
		if keyErr != nil {
			return keyErr
		}
		return fmt.Errorf("insert plan decision: %w", err)
	}
	return nil
}

func reusedDecisionKey(ctx context.Context, tx *sql.Tx, decision domain.PlanDecision) error {
	if decision.IdemKey == "" {
		return nil
	}
	var exists int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM external_plan_decisions
		WHERE work_id = ? AND idem_key = ?`, decision.WorkID, decision.IdemKey).Scan(&exists)
	if err == nil {
		return domain.ErrKeyReused
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check reused plan decision key: %w", err)
	}
	return nil
}

func insertPendingHandoff(ctx context.Context, tx *sql.Tx, in application.PlanDecisionWrite) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO external_plan_handoffs
		(id, decision_id, work_id, state, attempts, next_attempt_at)
		VALUES (?, ?, ?, 'pending', 0, ?)`, in.HandoffID, in.Decision.ID,
		in.Decision.WorkID, in.Decision.DecidedAt.Unix())
	if err != nil {
		return fmt.Errorf("insert plan handoff: %w", err)
	}
	return nil
}

func patchReviewedTask(ctx context.Context, tx *sql.Tx, in application.PlanDecisionWrite) error {
	status := domain.StatusWorking
	if in.Decision.Kind == domain.PlanDecisionApproved {
		status = domain.StatusDone
	}
	result, err := tx.ExecContext(ctx, `UPDATE tasks SET status = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND project_id = ? AND archived_at IS NULL`,
		string(status), in.Decision.DecidedAt.Unix(), in.Decision.TaskID,
		in.Decision.UserID, in.Decision.ProjectID)
	if err != nil {
		return fmt.Errorf("update reviewed task status: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count reviewed task update: %w", err)
	}
	if changed != 1 {
		return domain.ErrUnknownWork
	}
	return nil
}

func updateReviewEvent(ctx context.Context, tx *sql.Tx, in application.PlanDecisionWrite, eventID string) error {
	event, err := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+
		` FROM task_events WHERE task_id = ? AND id = ?`, in.Decision.TaskID, eventID))
	if err != nil {
		return err
	}
	if event.Kind != domain.EventPlanReview || event.PlanReview == nil || event.PlanReview.Revision != in.Revision {
		return domain.ErrStaleRevision
	}
	event.PlanReview.Status = domain.PlanRevisionStatus(in.Decision.Kind)
	event.PlanReview.Decision = string(in.Decision.Kind)
	event.PlanReview.Source = in.Decision.Source
	event.PlanReview.DecidedAt = in.Decision.DecidedAt
	event.PlanReview.HandoffID = in.HandoffID
	event.PlanReview.TaskID = ""
	return updateEventIn(ctx, tx, in.Decision.TaskID, event)
}

func insertDecisionMessage(ctx context.Context, tx *sql.Tx, ids application.IDs,
	in application.PlanDecisionWrite,
) error {
	text := "Plan approved. The Orchestrator is preparing the handoff."
	if in.Decision.Kind == domain.PlanDecisionChangesRequested {
		text = "Changes requested. The plan is back with the planner."
		if in.Decision.Note != "" {
			text += "\n\n" + in.Decision.Note
		}
	}
	_, err := insertEvent(ctx, tx, ids, in.Decision.TaskID, domain.Event{
		Kind: domain.EventMessage, Text: text, Time: in.Decision.DecidedAt,
	})
	return err
}
