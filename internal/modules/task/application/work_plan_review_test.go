package application

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func submitTestPlan(t *testing.T, works *Works, workID string) (PlanResult, SubmitPlanResult) {
	t.Helper()
	ctx := context.Background()
	plan, err := works.UpdatePlan(ctx, "u1", "ChatGPT", "proj", workID,
		UpdatePlanInput{Steps: []PlanStepInput{{Title: "Build"}}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if saveErr := works.SaveRequirements(ctx, "u1", "ChatGPT", "proj", workID, "Requirements"); saveErr != nil {
		t.Fatal(saveErr)
	}
	saveSourcedResearch(t, works, workID)
	submitted, err := works.SubmitPlan(ctx, "u1", "ChatGPT", "proj", workID,
		SubmitPlanInput{BaseRevision: plan.Revision, Summary: "Ready to review"})
	if err != nil || submitted.Revision.Status != domain.PlanRevisionSubmitted {
		t.Fatalf("submit plan: %+v, %v", submitted, err)
	}
	return plan, submitted
}

func assertPlanSubmitted(t *testing.T, store *workFakeStore, works *Works, workID string,
	submitted SubmitPlanResult,
) {
	t.Helper()
	if got := store.tasks[store.works[workID].TaskID].Status; got != domain.StatusNeedsInput {
		t.Fatalf("submitted plan status = %s", got)
	}
	assertTaskWorkSummary(t, works, store, workID)
	if submitted.Revision.RevNo != 1 {
		t.Fatalf("submitted revision = %d", submitted.Revision.RevNo)
	}
}

func assertPlanRetry(t *testing.T, works *Works, workID string, first SubmitPlanResult) {
	t.Helper()
	replay, err := works.SubmitPlan(context.Background(), "u1", "ChatGPT", "proj", workID,
		SubmitPlanInput{BaseRevision: 1, Summary: "Ready to review"})
	if err != nil || replay.Revision.ID != first.Revision.ID {
		t.Fatalf("identical submit did not reuse revision: %+v, %v", replay, err)
	}
}

func editAndAssertSuperseded(t *testing.T, works *Works, store *workFakeStore, workID string, revision int64) {
	t.Helper()
	if _, err := works.UpdatePlan(context.Background(), "u1", "ChatGPT", "proj", workID,
		UpdatePlanInput{BaseRevision: revision, Steps: []PlanStepInput{{Title: "Build better"}}}, ""); err != nil {
		t.Fatal(err)
	}
	latest, found, err := store.LatestPlan(context.Background(), workID)
	if err != nil || !found || latest.Status != domain.PlanRevisionSuperseded {
		t.Fatalf("edit did not supersede revision: %+v, %v", latest, err)
	}
	assertSupersededReviewEvent(t, store, workID)
}

func assertSupersededReviewEvent(t *testing.T, store *workFakeStore, workID string) {
	t.Helper()
	var card *domain.Event
	for index := range store.events[store.works[workID].TaskID] {
		event := &store.events[store.works[workID].TaskID][index]
		if event.Kind == domain.EventPlanReview {
			card = event
		}
	}
	if card == nil || card.PlanReview.Status != domain.PlanRevisionSuperseded || card.Version < 2 {
		t.Fatalf("review card was not updated in place: %+v", card)
	}
}
