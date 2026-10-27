package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestUpdatePlanRetryAndKeyMismatch(t *testing.T) {
	works, store, begun := begunWork(t)
	ctx := context.Background()
	in := UpdatePlanInput{Steps: []PlanStepInput{{Title: "Create page"}}}
	first, err := works.UpdatePlan(ctx, "u1", "ChatGPT", "proj", begun.WorkID, in, "request-001")
	if err != nil || first.Revision != 1 || first.Steps[0].ID == "" {
		t.Fatalf("first update: %+v, %v", first, err)
	}
	replay, err := works.UpdatePlan(ctx, "u1", "ChatGPT", "proj", begun.WorkID, in, "request-001")
	if err != nil || replay.Steps[0].ID != first.Steps[0].ID ||
		len(store.events[store.works[begun.WorkID].TaskID]) != 3 {
		t.Fatalf("retry duplicated or changed plan: %+v, %v", replay, err)
	}
	changed := UpdatePlanInput{Steps: []PlanStepInput{{Title: "Different request"}}}
	_, keyErr := works.UpdatePlan(ctx, "u1", "ChatGPT", "proj", begun.WorkID, changed, "request-001")
	if !errors.Is(keyErr, domain.ErrKeyReused) {
		t.Fatalf("reused request id with changed parameters was accepted: %v", keyErr)
	}
}

func TestPlanSaveStaysDraftUntilSubmitted(t *testing.T) {
	works, store, begun := begunWork(t)
	if _, err := works.UpdatePlan(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		UpdatePlanInput{Steps: []PlanStepInput{{Title: "Build"}}}, ""); err != nil {
		t.Fatal(err)
	}
	if work := store.works[begun.WorkID]; work.PlanState != domain.PlanDraft {
		t.Fatalf("saving plan advanced lifecycle: %s", work.PlanState)
	}
	if got := store.tasks[store.works[begun.WorkID].TaskID].Status; got == domain.StatusNeedsInput {
		t.Fatalf("saving plan changed task status: %s", got)
	}
}

func TestSubmitPlanAndLaterEditSupersedesReview(t *testing.T) {
	works, store, begun := begunWork(t)
	plan, submitted := submitTestPlan(t, works, begun.WorkID)
	assertPlanSubmitted(t, store, works, begun.WorkID, submitted)
	assertPlanRetry(t, works, begun.WorkID, submitted)
	editAndAssertSuperseded(t, works, store, begun.WorkID, plan.Revision)
}

func TestSubmitPlanRequiresRequirementsUnlessSmallTask(t *testing.T) {
	works, _, begun := begunWork(t)
	plan, err := works.UpdatePlan(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		UpdatePlanInput{Steps: []PlanStepInput{{Title: "Fix typo"}}}, "")
	if err != nil {
		t.Fatal(err)
	}
	saveSourcedResearch(t, works, begun.WorkID)
	input := SubmitPlanInput{BaseRevision: plan.Revision, Summary: "Fix one typo"}
	if !submitPlanErrorIs(t, works, begun.WorkID, input, domain.ErrBadRequest) {
		t.Fatal("plan without requirements was accepted")
	}
	input.SmallTask = true
	if _, err := works.SubmitPlan(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID, input); err != nil {
		t.Fatalf("small task could not submit without requirements: %v", err)
	}
}

func TestSmallTaskCanSubmitWithoutResearchOrRequirements(t *testing.T) {
	works, _, begun := begunWork(t)
	plan, err := works.UpdatePlan(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		UpdatePlanInput{Steps: []PlanStepInput{{Title: "Fix typo"}}}, "")
	if err != nil {
		t.Fatal(err)
	}
	_, missing, err := works.FinishResearch(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID)
	if err != nil || len(missing) == 0 {
		t.Fatalf("thin research check = %v, %v", missing, err)
	}
	_, err = works.SubmitPlan(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		SubmitPlanInput{BaseRevision: plan.Revision, Summary: "Fix one typo"})
	if !errors.Is(err, domain.ErrNoResearch) {
		t.Fatalf("plan without research = %v", err)
	}
	_, err = works.SubmitPlan(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		SubmitPlanInput{BaseRevision: plan.Revision, Summary: "Fix one typo", SmallTask: true})
	if err != nil {
		t.Fatalf("small task did not bypass research and requirements: %v", err)
	}
}

func saveSourcedResearch(t *testing.T, works *Works, workID string) {
	t.Helper()
	content := strings.Repeat("The source describes observed behavior and the relevant constraints. ", 8)
	if err := works.SaveResearch(context.Background(), "u1", "ChatGPT", "proj", workID,
		content, []string{"https://example.com/research"}); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteCannotFinishWhilePlanNeedsPerson(t *testing.T) {
	works, store, begun := begunWork(t)
	store.works[begun.WorkID] = domain.ExternalWork{
		ID: begun.WorkID, TaskID: store.works[begun.WorkID].TaskID,
		UserID: "u1", Client: "ChatGPT", ProjectID: "proj", State: domain.WorkOpen,
		PlanState: domain.PlanReadyForApproval,
	}
	_, err := works.Complete(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		"completed", "Done", nil)
	if !errors.Is(err, domain.ErrAwaitingPerson) {
		t.Fatalf("completion was accepted while waiting on person: %v", err)
	}
	_, err = works.Complete(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		"failed", "Stopped", nil)
	if err != nil {
		t.Fatalf("failure should remain allowed: %v", err)
	}
}

func TestUpdatePlanKeepsIDsAndReturnsCurrentOnStaleRevision(t *testing.T) {
	works, store, begun := begunWork(t)
	ctx := context.Background()
	first, err := works.UpdatePlan(ctx, "u1", "ChatGPT", "proj", begun.WorkID,
		UpdatePlanInput{Steps: []PlanStepInput{{Title: "Create page"}}}, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := works.UpdatePlan(ctx, "u1", "ChatGPT", "proj", begun.WorkID,
		UpdatePlanInput{BaseRevision: 1, Steps: []PlanStepInput{{Title: "Create page", Detail: "Use existing shell"}}}, "")
	if err != nil || second.Steps[0].ID != first.Steps[0].ID || second.Revision != 2 {
		t.Fatalf("step identity was not retained: %+v, %v", second, err)
	}
	stale, err := works.UpdatePlan(ctx, "u1", "ChatGPT", "proj", begun.WorkID,
		UpdatePlanInput{Steps: []PlanStepInput{{Title: "Outdated"}}}, "")
	if !errors.Is(err, domain.ErrStaleRevision) || stale.Revision != 2 || stale.Steps[0].Title != "Create page" {
		t.Fatalf("stale update did not return current plan: %+v, %v", stale, err)
	}
	assertTaskWorkSummary(t, works, store, begun.WorkID)
}

func assertTaskWorkSummary(t *testing.T, works *Works, store *workFakeStore, workID string) {
	t.Helper()
	clock := time.Now()
	service := NewService(testDeps(store, &clock), nil)
	service.UseWorkServices(works, store)
	task, err := service.Get(context.Background(), "u1", store.works[workID].TaskID)
	if err != nil || task.Work == nil || task.Work.ID != workID || task.Work.Revision != store.works[workID].Rev {
		t.Fatalf("task response did not include work summary: %+v, %v", task.Work, err)
	}
}
