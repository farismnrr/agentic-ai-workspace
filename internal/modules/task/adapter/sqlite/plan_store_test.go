package sqlite

import (
	"context"
	"database/sql"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestPlanRevisionIsImmutableAndReviewEventUpdatesInPlace(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-plan", "t-plan", "")
	markPlanRevision(t, ctx, db, work)
	revision := submitImmutableRevision(t, ctx, store, work)
	assertPlanRevisionImmutable(t, ctx, db, revision.ID)
	supersedeSubmittedRevision(t, ctx, store, work)
	assertSupersededReview(t, ctx, store, work.ID, work.TaskID, revision.EventID)
}

func markPlanRevision(t *testing.T, ctx context.Context, db *sql.DB, work domain.ExternalWork) {
	t.Helper()
	if _, err := db.ExecContext(ctx, `UPDATE external_works SET plan_rev = 1 WHERE id = ?`, work.ID); err != nil {
		t.Fatal(err)
	}
}

func submitImmutableRevision(t *testing.T, ctx context.Context, store *Store,
	work domain.ExternalWork,
) domain.PlanRevision {
	t.Helper()
	revision, err := store.SubmitPlan(ctx, application.PlanSubmission{
		WorkID: work.ID, UserID: work.UserID, ExpectRev: work.Rev,
		Revision: domain.PlanRevision{
			ID: "revision-1", WorkID: work.ID, TaskID: work.TaskID,
			PlanRev: 1, Steps: []domain.PlanStep{{ID: "one", Title: "Build"}}, Summary: "Review this", Digest: "digest",
		},
		Event: domain.Event{
			Kind: domain.EventPlanReview, Text: "Plan ready for approval",
			PlanReview: &domain.PlanReview{Summary: "Review this", Status: domain.PlanRevisionSubmitted},
		},
	})
	if err != nil || revision.RevNo != 1 || revision.EventID == "" {
		t.Fatalf("submit revision: %+v, %v", revision, err)
	}
	return revision
}

func assertPlanRevisionImmutable(t *testing.T, ctx context.Context, db *sql.DB, revisionID string) {
	t.Helper()
	_, updateErr := db.ExecContext(ctx,
		`UPDATE external_plan_revisions SET summary = 'changed' WHERE id = ?`, revisionID)
	if updateErr == nil {
		t.Fatal("revision content update was allowed")
	}
}

func supersedeSubmittedRevision(t *testing.T, ctx context.Context, store *Store, work domain.ExternalWork) {
	t.Helper()
	change := application.WorkChange{
		WorkID: work.ID, UserID: work.UserID, ExpectRev: 1,
		Patch: &application.WorkPatch{PlanState: phasePtr(domain.PlanDraft), SupersedePlan: true},
	}
	if _, _, commitErr := store.Commit(ctx, change); commitErr != nil {
		t.Fatal(commitErr)
	}
}

func assertSupersededReview(t *testing.T, ctx context.Context, store *Store,
	workID, taskID, eventID string,
) {
	t.Helper()
	latest, found, err := store.LatestPlan(ctx, workID)
	if err != nil || !found || latest.Status != domain.PlanRevisionSuperseded {
		t.Fatalf("revision status = %+v found=%v err=%v", latest, found, err)
	}
	event, err := store.FindEvent(ctx, taskID, eventID)
	if err != nil || event.PlanReview.Status != domain.PlanRevisionSuperseded || event.Version < 2 {
		t.Fatalf("review event did not update in place: %+v, %v", event, err)
	}
}

func phasePtr(value domain.PlanPhase) *domain.PlanPhase { return &value }
