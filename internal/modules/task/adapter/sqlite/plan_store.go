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

// SubmitPlan atomically changes phase/status and persists the immutable snapshot and card event.
func (s *Store) SubmitPlan(ctx context.Context, in application.PlanSubmission) (domain.PlanRevision, error) {
	return s.submitPlanSnapshot(ctx, in.WorkID, in.UserID, in.ExpectRev, in.Revision, in.Event)
}

// LatestPlan returns the newest persisted revision, whether it is submitted or superseded.
func (s *Store) LatestPlan(ctx context.Context, workID string) (domain.PlanRevision, bool, error) {
	return s.planRevision(ctx, workID, 0)
}

// PlanRevision reads one immutable submitted snapshot.
func (s *Store) PlanRevision(ctx context.Context, workID string, revision int64) (domain.PlanRevision, bool, error) {
	return s.planRevision(ctx, workID, revision)
}

func (s *Store) planRevision(ctx context.Context, workID string, revisionNo int64) (domain.PlanRevision, bool, error) {
	var revision domain.PlanRevision
	var steps string
	var smallTask int
	var created int64
	query := `SELECT id, work_id, task_id, rev_no, plan_rev, steps, prd, research, qa,
		small_task, summary, digest, event_id, status, created_at FROM external_plan_revisions WHERE work_id = ?`
	args := []any{workID}
	if revisionNo > 0 {
		query += ` AND rev_no = ?`
		args = append(args, revisionNo)
	} else {
		query += ` ORDER BY rev_no DESC LIMIT 1`
	}
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&revision.ID, &revision.WorkID, &revision.TaskID, &revision.RevNo, &revision.PlanRev, &steps,
		&revision.PRD, &revision.Research, &revision.QA, &smallTask, &revision.Summary, &revision.Digest,
		&revision.EventID, &revision.Status, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PlanRevision{}, false, nil
	}
	if err != nil {
		return domain.PlanRevision{}, false, fmt.Errorf("read latest plan revision: %w", err)
	}
	if err := json.Unmarshal([]byte(steps), &revision.Steps); err != nil {
		return domain.PlanRevision{}, false, fmt.Errorf("decode latest plan revision: %w", err)
	}
	revision.SmallTask, revision.CreatedAt = smallTask != 0, unixTime(created)
	return revision, true, nil
}

// LatestPlanReview reads the durable decision and handoff paired with the latest immutable revision.
func (s *Store) LatestPlanReview(ctx context.Context, workID string) (domain.PlanReview, bool, error) {
	var review domain.PlanReview
	var smallTask int
	var decidedAt int64
	err := s.db.QueryRowContext(ctx, `SELECT r.rev_no, r.summary, r.status, r.small_task, r.task_id,
		COALESCE(d.decision, ''), COALESCE(h.id, ''), COALESCE(h.state, ''),
		COALESCE(h.internal_task_id, ''), COALESCE(d.source, ''), COALESCE(d.decided_at, 0)
		FROM external_plan_revisions r
		LEFT JOIN external_plan_decisions d ON d.revision_id = r.id
		LEFT JOIN external_plan_handoffs h ON h.decision_id = d.id
		WHERE r.work_id = ? ORDER BY r.rev_no DESC LIMIT 1`, workID).Scan(
		&review.Revision, &review.Summary, &review.Status, &smallTask, &review.TaskID, &review.Decision,
		&review.HandoffID, &review.HandoffState, &review.LinkedTaskID, &review.Source, &decidedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PlanReview{}, false, nil
	}
	if err != nil {
		return domain.PlanReview{}, false, fmt.Errorf("read latest plan review: %w", err)
	}
	review.SmallTask = smallTask != 0
	if decidedAt > 0 {
		review.DecidedAt = unixTime(decidedAt)
	}
	return review, true, nil
}

var _ application.PlanStore = (*Store)(nil)

func unixTime(value int64) time.Time { return time.Unix(value, 0).UTC() }
