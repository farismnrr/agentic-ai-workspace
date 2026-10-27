package application

import (
	"context"
	"encoding/json"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// PlanStepInput is one proposed step in an external plan.
type PlanStepInput struct {
	ID     string           `json:"id,omitempty"`
	Title  string           `json:"title"`
	Detail string           `json:"detail,omitempty"`
	State  domain.PlanState `json:"state,omitempty"`
}

// UpdatePlanInput replaces the current plan if its revision still matches.
type UpdatePlanInput struct {
	Steps        []PlanStepInput `json:"steps"`
	BaseRevision int64           `json:"base_revision"`
}

// PlanResult reports the persisted plan and its revision, including on a stale write.
type PlanResult struct {
	Steps    []domain.PlanStep `json:"steps"`
	Revision int64             `json:"revision"`
}

// UpdatePlan reconciles stable step IDs and protects concurrent edits with a plan revision.
func (w *Works) UpdatePlan(ctx context.Context, userID, client, projectID, workID string,
	in UpdatePlanInput, requestID string,
) (PlanResult, error) {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return PlanResult{}, err
	}
	if !validKey(requestID) {
		return PlanResult{}, domain.ErrBadRequest
	}
	cleanInput, err := normalizePlanInput(in)
	if err != nil {
		return PlanResult{}, err
	}
	hash := planRequestHash(in.BaseRevision, cleanInput)
	_, replay, err := w.planReplay(ctx, userID, client, projectID, workID, requestID, hash)
	if err != nil {
		return PlanResult{}, err
	}
	if replay != nil {
		return *replay, nil
	}
	work, task, resumed, err := w.mutable(ctx, userID, client, projectID, workID)
	if err != nil {
		return PlanResult{}, err
	}
	current := PlanResult{Steps: task.Plan, Revision: work.PlanRev}
	if in.BaseRevision != work.PlanRev {
		return current, domain.ErrStaleRevision
	}
	return w.savePlan(ctx, work, task, resumed, cleanInput, requestID, hash)
}

func (w *Works) planReplay(ctx context.Context, userID, client, projectID, workID, key, hash string) (
	domain.ExternalWork, *PlanResult, error,
) {
	work, _, err := w.loadWorkTask(ctx, userID, client, projectID, workID)
	if err != nil {
		return domain.ExternalWork{}, nil, err
	}
	result, ok, err := w.replay(ctx, work, "update_plan", key, hash)
	if err != nil {
		return domain.ExternalWork{}, nil, err
	}
	if !ok {
		return work, nil, nil
	}
	return work, &result, nil
}

func (w *Works) savePlan(ctx context.Context, work domain.ExternalWork, task domain.Task, resumed bool,
	incoming []domain.PlanStep, requestID, hash string,
) (PlanResult, error) {
	current := PlanResult{Steps: task.Plan, Revision: work.PlanRev}
	steps, diff := domain.ReconcilePlan(task.Plan, incoming)
	if !diff.Changed && !resumed && requestID == "" {
		return current, nil
	}
	nextRevision := work.PlanRev
	technical, _ := json.Marshal(diff)
	patch := &WorkPatch{}
	taskPatch := &TaskPatch{}
	events := []domain.Event{}
	if diff.Changed {
		nextRevision++
		patch.PlanRev = &nextRevision
		taskPatch.Plan = &steps
		events = append(events, workEvent(work, domain.EventPlan, "Plan updated", string(technical), w.now()))
	}
	if resumed {
		events = append(events, resumedEvent(work, w.now()))
		taskPatch.Status = statusPtr(domain.StatusWorking)
		patch.State = workStatePtr(domain.WorkOpen)
	}
	result := PlanResult{Steps: steps, Revision: nextRevision}
	if err := w.commitPlan(ctx, work, requestID, hash, result, patch, taskPatch, events); err != nil {
		return PlanResult{}, err
	}
	if diff.Changed || resumed {
		w.syncWorkNotes(ctx, work)
		w.notify(ctx, domain.Task{ID: work.TaskID, UserID: work.UserID})
	}
	return PlanResult{Steps: steps, Revision: nextRevision}, nil
}

func planRequestHash(baseRevision int64, steps []domain.PlanStep) string {
	return requestHash(struct {
		Steps        []domain.PlanStep
		BaseRevision int64
	}{steps, baseRevision})
}
