package application

import (
	"context"
	"errors"
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
		len(store.events[store.works[begun.WorkID].TaskID]) != 2 {
		t.Fatalf("retry duplicated or changed plan: %+v, %v", replay, err)
	}
	changed := UpdatePlanInput{Steps: []PlanStepInput{{Title: "Different request"}}}
	_, keyErr := works.UpdatePlan(ctx, "u1", "ChatGPT", "proj", begun.WorkID, changed, "request-001")
	if !errors.Is(keyErr, domain.ErrKeyReused) {
		t.Fatalf("reused request id with changed parameters was accepted: %v", keyErr)
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
