package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// SavePlan replaces a task's plan.
func (s *Store) SavePlan(ctx context.Context, id string, plan []domain.PlanStep, at time.Time) error {
	encoded, err := json.Marshal(append([]domain.PlanStep{}, plan...))
	if err != nil {
		return fmt.Errorf("encode plan: %w", err)
	}

	return s.exec(ctx, `UPDATE tasks SET plan = ?, updated_at = ? WHERE id = ?`, string(encoded), at.Unix(), id)
}

// SavePRD records the product requirements document the team builds from.
func (s *Store) SavePRD(ctx context.Context, id, prd string, at time.Time) error {
	return s.exec(ctx, `UPDATE tasks SET prd = ?, updated_at = ? WHERE id = ?`, prd, at.Unix(), id)
}

// SaveResearch records the research notes the team builds from.
func (s *Store) SaveResearch(ctx context.Context, id, research string, at time.Time) error {
	return s.exec(ctx, `UPDATE tasks SET research = ?, updated_at = ? WHERE id = ?`, research, at.Unix(), id)
}

// Finish hands a task to review with its summary and proof.
func (s *Store) Finish(ctx context.Context, id, summary string, proof *domain.Proof, at time.Time) error {
	encoded, err := json.Marshal(proof)
	if err != nil {
		return fmt.Errorf("encode proof: %w", err)
	}

	return s.exec(ctx, `UPDATE tasks SET status = 'review', summary = ?, proof = ?, updated_at = ? WHERE id = ? AND status <> 'done'`,
		summary, string(encoded), at.Unix(), id)
}

// CompleteExternalLead atomically records an external lead's unverified report.
func (s *Store) CompleteExternalLead(
	ctx context.Context, id, source, summary string, at time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin external lead completion: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var status, currentSummary, currentSource string
	err = tx.QueryRowContext(ctx, `SELECT status, summary, source FROM tasks WHERE id = ?`, id).
		Scan(&status, &currentSummary, &currentSource)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrWrongState
	}
	if err != nil {
		return fmt.Errorf("read external lead task: %w", err)
	}
	if currentSource != source {
		return domain.ErrWrongState
	}
	if status == string(domain.StatusDone) {
		return externalCompletionRetry(ctx, tx, id, source, currentSummary, summary)
	}
	if err := rejectPendingOwnerAction(ctx, tx, id); err != nil {
		return err
	}
	var activeJobs int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_delegations
		WHERE task_id = ? AND status IN ('working', 'waiting')`, id).Scan(&activeJobs); err != nil {
		return fmt.Errorf("check active external workers: %w", err)
	}
	if activeJobs > 0 {
		return errors.New("worker jobs are still active")
	}
	if status != string(domain.StatusWorking) {
		return domain.ErrWrongState
	}
	result, err := tx.ExecContext(ctx, `UPDATE tasks SET status = 'done', summary = ?, proof = NULL, updated_at = ?
		WHERE id = ? AND source = ? AND status = 'working'`, summary, at.Unix(), id, source)
	if err != nil {
		return fmt.Errorf("complete external lead task: %w", err)
	}
	if changed, rowsErr := result.RowsAffected(); rowsErr != nil || changed != 1 {
		if rowsErr != nil {
			return fmt.Errorf("count completed external lead task: %w", rowsErr)
		}
		return domain.ErrWrongState
	}
	_, err = insertEvent(ctx, tx, s.ids, id, domain.Event{
		Kind: domain.EventMessage, Text: summary, Technical: externalLeadReportMarker,
		Agent: source, Time: at,
	})
	if err != nil {
		return err
	}
	return tx.Commit()
}

const externalLeadReportMarker = "External lead report; not independently verified."

func externalCompletionRetry(
	ctx context.Context, tx *sql.Tx, id, source, currentSummary, summary string,
) error {
	if currentSummary != summary {
		return domain.ErrWrongState
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+eventColumns+` FROM task_events WHERE task_id = ?`, id)
	if err != nil {
		return fmt.Errorf("read external completion event: %w", err)
	}
	events, err := collectEvents(rows, "read external completion events")
	if err != nil {
		return err
	}
	for _, event := range events {
		if event.Kind == domain.EventMessage && event.Agent == source &&
			event.Text == summary && event.Technical == externalLeadReportMarker {
			return nil
		}
	}
	return domain.ErrWrongState
}

func rejectPendingOwnerAction(ctx context.Context, tx *sql.Tx, id string) error {
	rows, err := tx.QueryContext(ctx, `SELECT `+eventColumns+` FROM task_events WHERE task_id = ?`, id)
	if err != nil {
		return fmt.Errorf("read external task actions: %w", err)
	}
	events, err := collectEvents(rows, "read external task actions")
	if err != nil {
		return err
	}
	for _, event := range events {
		if event.Kind == domain.EventQuestion && event.Answered == "" ||
			event.Kind == domain.EventApproval && event.Resolved == "" {
			return domain.ErrNeedsAnswer
		}
	}
	return nil
}
