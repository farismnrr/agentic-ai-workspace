package sqlite

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestPlanRevisionIsImmutableAndReviewEventUpdatesInPlace(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-plan", "t-plan", "")
	if _, err := db.ExecContext(ctx, `UPDATE external_works SET plan_rev = 1 WHERE id = ?`, work.ID); err != nil {
		t.Fatal(err)
	}
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
	_, updateErr := db.ExecContext(ctx,
		`UPDATE external_plan_revisions SET summary = 'changed' WHERE id = ?`, revision.ID)
	if updateErr == nil {
		t.Fatal("revision content update was allowed")
	}
	change := application.WorkChange{
		WorkID: work.ID, UserID: work.UserID, ExpectRev: 1,
		Patch: &application.WorkPatch{PlanState: phasePtr(domain.PlanDraft), SupersedePlan: true},
	}
	if _, _, commitErr := store.Commit(ctx, change); commitErr != nil {
		t.Fatal(commitErr)
	}
	latest, found, err := store.LatestPlan(ctx, work.ID)
	if err != nil || !found || latest.Status != domain.PlanRevisionSuperseded {
		t.Fatalf("revision status = %+v found=%v err=%v", latest, found, err)
	}
	event, err := store.FindEvent(ctx, work.TaskID, revision.EventID)
	if err != nil || event.PlanReview.Status != domain.PlanRevisionSuperseded || event.Version < 2 {
		t.Fatalf("review event did not update in place: %+v, %v", event, err)
	}
}

func phasePtr(value domain.PlanPhase) *domain.PlanPhase { return &value }
