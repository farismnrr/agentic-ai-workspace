package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

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
	assertChangesDecision(t, result, err)
	assertPlanReturnedToDraft(t, ctx, store, db, work)
	assertNoChangesHandoff(t, ctx, db, work.ID)
}

func assertChangesDecision(t *testing.T, result application.PlanDecisionResult, err error) {
	t.Helper()
	if err != nil || result.Decision.Kind != domain.PlanDecisionChangesRequested || result.HandoffID != "" {
		t.Fatalf("request changes = %+v, %v", result, err)
	}
}

func assertPlanReturnedToDraft(t *testing.T, ctx context.Context, store *Store, db *sql.DB,
	work domain.ExternalWork,
) {
	t.Helper()
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
}

func assertNoChangesHandoff(t *testing.T, ctx context.Context, db *sql.DB, workID string) {
	t.Helper()
	var handoffs int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_plan_handoffs WHERE work_id = ?`, workID).
		Scan(&handoffs)
	if handoffs != 0 {
		t.Fatalf("request changes created %d handoffs", handoffs)
	}
}
