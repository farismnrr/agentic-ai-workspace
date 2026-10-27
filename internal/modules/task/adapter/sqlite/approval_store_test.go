package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestApprovePlanTransactionPersistsOneDecisionAndReplaysItsHandoff(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-approve", "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	in := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
	first := approveOnce(t, store, in)
	assertApprovalReplay(t, store, in, work, revision, first)
	assertLatestApprovalReview(t, store, work, first)
	assertReusedApprovalKeyRejected(t, store, in)
}

func approveOnce(t *testing.T, store *Store, in application.PlanDecisionWrite) application.PlanDecisionResult {
	t.Helper()
	first, err := store.DecidePlan(t.Context(), in)
	if err != nil || first.HandoffID != in.HandoffID || first.HandoffState != "pending" {
		t.Fatalf("approve plan = %+v, %v", first, err)
	}
	return first
}

func assertApprovalReplay(t *testing.T, store *Store, in application.PlanDecisionWrite,
	work domain.ExternalWork, revision domain.PlanRevision, first application.PlanDecisionResult,
) {
	t.Helper()
	second, err := store.DecidePlan(t.Context(), in)
	if err != nil || second.Decision.ID != first.Decision.ID || second.HandoffID != first.HandoffID {
		t.Fatalf("approval retry = %+v, %v", second, err)
	}
	assertApprovalRows(t, t.Context(), store.db, work, revision, first)
}

func assertLatestApprovalReview(t *testing.T, store *Store, work domain.ExternalWork,
	first application.PlanDecisionResult,
) {
	t.Helper()
	review, found, err := store.LatestPlanReview(t.Context(), work.ID)
	if err != nil || !found || review.Decision != string(domain.PlanDecisionApproved) ||
		review.HandoffID != first.HandoffID || review.HandoffState != "pending" {
		t.Fatalf("latest review state = %+v, found %v, error %v", review, found, err)
	}
}

func assertReusedApprovalKeyRejected(t *testing.T, store *Store, in application.PlanDecisionWrite) {
	t.Helper()
	reused := in
	reused.Revision++
	reused.Decision.ReqHash = "different-revision-hash"
	if _, err := store.DecidePlan(t.Context(), reused); !errors.Is(err, domain.ErrKeyReused) {
		t.Fatalf("same key for another revision error = %v", err)
	}
}

func assertApprovalRows(t *testing.T, ctx context.Context, db *sql.DB, work domain.ExternalWork,
	revision domain.PlanRevision, result application.PlanDecisionResult,
) {
	t.Helper()
	assertApprovedWorkRows(t, ctx, db, work.ID, work.TaskID)
	assertApprovedReviewRows(t, ctx, db, work.TaskID, revision.EventID, result.HandoffID)
	assertApprovalOutboxRows(t, ctx, db, work.ID)
}

func assertApprovedWorkRows(t *testing.T, ctx context.Context, db *sql.DB, workID, taskID string) {
	t.Helper()
	var workState, phase, taskStatus string
	if err := db.QueryRowContext(ctx, `SELECT state, plan_state FROM external_works WHERE id = ?`, workID).
		Scan(&workState, &phase); err != nil || workState != "completed" || phase != "approved" {
		t.Fatalf("approved work = %s/%s, %v", workState, phase, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, taskID).
		Scan(&taskStatus); err != nil || taskStatus != "done" {
		t.Fatalf("approved task status = %s, %v", taskStatus, err)
	}
}

func assertApprovedReviewRows(t *testing.T, ctx context.Context, db *sql.DB,
	taskID, eventID, handoffID string,
) {
	t.Helper()
	event, err := NewStore(db, &workTestIDs{}).FindEvent(ctx, taskID, eventID)
	if err != nil || event.PlanReview.Status != domain.PlanRevisionApproved ||
		event.PlanReview.HandoffID != handoffID ||
		event.Version < 2 {
		t.Fatalf("review card = %+v, %v", event.PlanReview, err)
	}
	events, err := NewStore(db, &workTestIDs{}).EventsAfter(ctx, taskID, 0)
	if err != nil || len(events) != 3 {
		t.Fatalf("decision should append exactly one message event, events=%d err=%v", len(events), err)
	}
}

func assertApprovalOutboxRows(t *testing.T, ctx context.Context, db *sql.DB, workID string) {
	t.Helper()
	var decisions, handoffs int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_decisions WHERE work_id = ?`, workID).
		Scan(&decisions); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_handoffs WHERE work_id = ?`, workID).
		Scan(&handoffs); err != nil || decisions != 1 || handoffs != 1 {
		t.Fatalf("approval/outbox counts = %d/%d, %v", decisions, handoffs, err)
	}
}

func TestPlanApprovalRejectsWrongOwnerAndStaleRevision(t *testing.T) {
	for _, test := range []struct {
		want error
		edit func(*application.PlanDecisionWrite)
		name string
	}{
		{
			name: "wrong owner", edit: func(in *application.PlanDecisionWrite) { in.Decision.UserID = "other" },
			want: domain.ErrUnknownWork,
		},
		{
			name: "wrong project", edit: func(in *application.PlanDecisionWrite) { in.Decision.ProjectID = "other" },
			want: domain.ErrUnknownWork,
		},
		{
			name: "stale revision", edit: func(in *application.PlanDecisionWrite) { in.Revision = 2 },
			want: domain.ErrStaleRevision,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, _, store := newWorkStore(t)
			work := createTestWork(t, ctx, store, "w-reject", "t1", "")
			revision := submitStorePlan(t, ctx, store, work)
			in := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
			test.edit(&in)
			if _, err := store.DecidePlan(ctx, in); !errors.Is(err, test.want) {
				t.Fatalf("approval error = %v, want %v", err, test.want)
			}
			latest, found, err := store.LatestPlan(ctx, work.ID)
			if err != nil || !found || latest.Status != domain.PlanRevisionSubmitted {
				t.Fatalf("rejected approval changed revision: %+v %v", latest, err)
			}
		})
	}
}
