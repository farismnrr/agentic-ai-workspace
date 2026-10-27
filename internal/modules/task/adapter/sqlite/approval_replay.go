package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func persistedDecision(ctx context.Context, tx *sql.Tx, in application.PlanDecisionWrite) (
	application.PlanDecisionResult, error,
) {
	result, err := readDecision(ctx, tx, `d.work_id = ? AND d.idem_key = ? AND d.idem_key <> ''
		AND d.user_id = ? AND d.project_id = ? AND d.task_id = ?`,
		in.Decision.WorkID, in.Decision.IdemKey, in.Decision.UserID, in.Decision.ProjectID, in.Decision.TaskID)
	if err == nil {
		if result.Decision.ReqHash != in.Decision.ReqHash {
			return application.PlanDecisionResult{}, domain.ErrKeyReused
		}
		return rearmFailedDecision(ctx, tx, in, result)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return application.PlanDecisionResult{}, fmt.Errorf("replay plan decision: %w", err)
	}
	result, err = readDecision(ctx, tx, `d.work_id = ? AND r.rev_no = ? AND d.user_id = ?
		AND d.project_id = ? AND d.task_id = ?`,
		in.Decision.WorkID, in.Revision, in.Decision.UserID, in.Decision.ProjectID, in.Decision.TaskID)
	if err == nil {
		if result.Decision.Kind != in.Decision.Kind {
			return application.PlanDecisionResult{}, domain.ErrStaleRevision
		}
		return rearmFailedDecision(ctx, tx, in, result)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return application.PlanDecisionResult{}, err
	}
	if err := checkDecisionOwner(ctx, tx, in); err != nil {
		return application.PlanDecisionResult{}, err
	}
	return application.PlanDecisionResult{}, domain.ErrStaleRevision
}

func readDecision(ctx context.Context, tx *sql.Tx, predicate string, args ...any) (
	application.PlanDecisionResult, error,
) {
	query := `SELECT d.id, d.revision_id, d.work_id, d.task_id, d.user_id, d.project_id, d.decision,
		d.source, d.client, d.idem_key, d.req_hash, d.note, d.decided_at, COALESCE(h.id, ''),
		COALESCE(h.state, ''), COALESCE(h.internal_task_id, '') FROM external_plan_decisions d
		JOIN external_plan_revisions r ON r.id = d.revision_id LEFT JOIN external_plan_handoffs h
		ON h.decision_id = d.id WHERE ` + predicate
	var result application.PlanDecisionResult
	var kind string
	var decidedAt int64
	err := tx.QueryRowContext(ctx, query, args...).Scan(
		&result.Decision.ID, &result.Decision.RevisionID, &result.Decision.WorkID, &result.Decision.TaskID,
		&result.Decision.UserID, &result.Decision.ProjectID, &kind, &result.Decision.Source, &result.Decision.Client,
		&result.Decision.IdemKey, &result.Decision.ReqHash, &result.Decision.Note, &decidedAt,
		&result.HandoffID, &result.HandoffState, &result.InternalTaskID)
	if err != nil {
		return application.PlanDecisionResult{}, fmt.Errorf("read replayed plan decision: %w", err)
	}
	result.Decision.Kind, result.Decision.DecidedAt = domain.PlanDecisionKind(kind), unixTime(decidedAt)
	return result, nil
}

func rearmFailedDecision(ctx context.Context, tx *sql.Tx, in application.PlanDecisionWrite,
	result application.PlanDecisionResult,
) (application.PlanDecisionResult, error) {
	if result.Decision.Kind != domain.PlanDecisionApproved || result.HandoffState != "failed" {
		return result, nil
	}
	_, err := tx.ExecContext(ctx, `UPDATE external_plan_handoffs SET state = 'pending', attempts = 0,
		next_attempt_at = ?, last_error = '' WHERE decision_id = ? AND state = 'failed'`,
		in.Decision.DecidedAt.Unix(), result.Decision.ID)
	if err != nil {
		return application.PlanDecisionResult{}, mapBusy(fmt.Errorf("rearm failed plan handoff: %w", err))
	}
	result.HandoffState = "pending"
	result.InternalTaskID = ""
	return result, nil
}

func checkDecisionOwner(ctx context.Context, tx *sql.Tx, in application.PlanDecisionWrite) error {
	var exists int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM external_works WHERE id = ? AND task_id = ? AND user_id = ?
		AND project_id = ? AND (? != 'mcp' OR client = ?)`, in.Decision.WorkID, in.Decision.TaskID,
		in.Decision.UserID, in.Decision.ProjectID, in.Decision.Source, in.Decision.Client).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrUnknownWork
	}
	if err != nil {
		return fmt.Errorf("check plan decision owner: %w", err)
	}
	return nil
}
