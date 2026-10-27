package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestApprovePlanTransactionPersistsOneDecisionAndReplaysItsHandoff(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-approve", "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	in := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
	first, err := store.DecidePlan(ctx, in)
	if err != nil || first.HandoffID != in.HandoffID || first.HandoffState != "pending" {
		t.Fatalf("approve plan = %+v, %v", first, err)
	}
	second, err := store.DecidePlan(ctx, in)
	if err != nil || second.Decision.ID != first.Decision.ID || second.HandoffID != first.HandoffID {
		t.Fatalf("approval retry = %+v, %v", second, err)
	}
	assertApprovalRows(t, ctx, db, work, revision, first)
	review, found, err := store.LatestPlanReview(ctx, work.ID)
	if err != nil || !found || review.Decision != string(domain.PlanDecisionApproved) ||
		review.HandoffID != first.HandoffID || review.HandoffState != "pending" {
		t.Fatalf("latest review state = %+v, found %v, error %v", review, found, err)
	}

	reused := in
	reused.Revision++
	reused.Decision.ReqHash = "different-revision-hash"
	if _, err := store.DecidePlan(ctx, reused); !errors.Is(err, domain.ErrKeyReused) {
		t.Fatalf("same key for another revision error = %v", err)
	}
}

func assertApprovalRows(t *testing.T, ctx context.Context, db *sql.DB, work domain.ExternalWork,
	revision domain.PlanRevision, result application.PlanDecisionResult,
) {
	t.Helper()
	var workState, phase, taskStatus string
	if err := db.QueryRowContext(ctx, `SELECT state, plan_state FROM external_works WHERE id = ?`, work.ID).
		Scan(&workState, &phase); err != nil || workState != "completed" || phase != "approved" {
		t.Fatalf("approved work = %s/%s, %v", workState, phase, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, work.TaskID).
		Scan(&taskStatus); err != nil || taskStatus != "done" {
		t.Fatalf("approved task status = %s, %v", taskStatus, err)
	}
	event, err := NewStore(db, &workTestIDs{}).FindEvent(ctx, work.TaskID, revision.EventID)
	if err != nil || event.PlanReview.Status != domain.PlanRevisionApproved ||
		event.PlanReview.HandoffID != result.HandoffID ||
		event.Version < 2 {
		t.Fatalf("review card = %+v, %v", event.PlanReview, err)
	}
	events, err := NewStore(db, &workTestIDs{}).EventsAfter(ctx, work.TaskID, 0)
	if err != nil || len(events) != 3 {
		t.Fatalf("decision should append exactly one message event, events=%d err=%v", len(events), err)
	}
	var decisions, handoffs int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_decisions WHERE work_id = ?`, work.ID).
		Scan(&decisions); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_handoffs WHERE work_id = ?`, work.ID).
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

func TestRequestChangesStoresDecisionAndReturnsThePlanToDraft(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-changes", "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	in := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
	in.Decision.ID = "decision-changes"
	in.Decision.Kind = domain.PlanDecisionChangesRequested
	in.Decision.Source = "app"
	in.Decision.Client = ""
	in.Decision.IdemKey = "changes-key-123"
	in.Decision.ReqHash = "changes-hash"
	in.Decision.Note = "Clarify the first step."
	in.HandoffID = ""
	result, err := store.DecidePlan(ctx, in)
	if err != nil || result.Decision.Kind != domain.PlanDecisionChangesRequested || result.HandoffID != "" {
		t.Fatalf("request changes = %+v, %v", result, err)
	}
	latest, found, err := store.LatestPlan(ctx, work.ID)
	if err != nil || !found || latest.Status != domain.PlanRevisionChangesRequested {
		t.Fatalf("revision after changes requested = %+v, %v", latest, err)
	}
	var phase, state, taskStatus string
	_ = db.QueryRowContext(ctx, `SELECT plan_state,state FROM external_works WHERE id = ?`, work.ID).
		Scan(&phase, &state)
	_ = db.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, work.TaskID).Scan(&taskStatus)
	if phase != string(domain.PlanDraft) || state != string(domain.WorkOpen) ||
		taskStatus != string(domain.StatusWorking) {
		t.Fatalf("changes requested left work in %s/%s with task %s", phase, state, taskStatus)
	}
	var handoffs int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_handoffs WHERE work_id = ?`, work.ID).Scan(&handoffs)
	if handoffs != 0 {
		t.Fatalf("request changes created %d handoffs", handoffs)
	}
}

func TestConcurrentApprovalCreatesOneDecisionAndHandoff(t *testing.T) {
	ctx, db, store := newConcurrentWorkStore(t)
	work := createTestWork(t, ctx, store, "w-race-approval", "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for index := range 2 {
		wait.Add(1)
		go func(number int) {
			defer wait.Done()
			in := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
			in.Decision.ID = fmt.Sprintf("decision-race-%d", number)
			in.Decision.IdemKey = fmt.Sprintf("race-key-%d", number)
			in.Decision.ReqHash = fmt.Sprintf("race-hash-%d", number)
			in.HandoffID = fmt.Sprintf("handoff-race-%d", number)
			_, err := store.DecidePlan(ctx, in)
			results <- err
		}(index)
	}
	wait.Wait()
	close(results)
	for err := range results {
		if err != nil && !errors.Is(err, domain.ErrTryAgain) {
			t.Fatalf("concurrent approval error = %v", err)
		}
	}
	var decisions, handoffs int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_decisions WHERE work_id = ?`, work.ID).
		Scan(&decisions)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_handoffs WHERE work_id = ?`, work.ID).
		Scan(&handoffs)
	if decisions != 1 || handoffs != 1 {
		t.Fatalf("concurrent approvals persisted %d decisions and %d handoffs", decisions, handoffs)
	}
}
