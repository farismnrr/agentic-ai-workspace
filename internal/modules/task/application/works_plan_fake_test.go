package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (f *workFakeStore) SubmitPlan(_ context.Context, in PlanSubmission) (domain.PlanRevision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	work, ok := f.works[in.WorkID]
	if !ok || work.UserID != in.UserID {
		return domain.PlanRevision{}, domain.ErrUnknownWork
	}
	if work.Rev != in.ExpectRev || work.PlanState != domain.PlanDraft {
		return domain.PlanRevision{}, domain.ErrStaleRevision
	}
	work.Rev++
	work.PlanState, work.UpdatedAt = domain.PlanReadyForApproval, time.Now()
	f.works[work.ID] = work
	task := f.tasks[work.TaskID]
	task.Status = domain.StatusNeedsInput
	f.tasks[task.ID] = task
	event := in.Event
	event.ID = "plan-review"
	event.Seq, event.Version = int64(len(f.events[task.ID])+1), int64(len(f.events[task.ID])+1)
	f.events[task.ID] = append(f.events[task.ID], event)
	revision := in.Revision
	revision.RevNo, revision.EventID = int64(len(f.plans[work.ID])+1), event.ID
	revision.Status = domain.PlanRevisionSubmitted
	f.plans[work.ID] = append(f.plans[work.ID], revision)
	return revision, nil
}

func (f *workFakeStore) LatestPlan(_ context.Context, workID string) (domain.PlanRevision, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	plans := f.plans[workID]
	if len(plans) == 0 {
		return domain.PlanRevision{}, false, nil
	}
	return plans[len(plans)-1], true, nil
}

func (f *workFakeStore) LatestPlanReview(ctx context.Context, workID string) (domain.PlanReview, bool, error) {
	revision, found, err := f.LatestPlan(ctx, workID)
	if err != nil || !found {
		return domain.PlanReview{}, found, err
	}
	return domain.PlanReview{
		Revision: revision.RevNo, Summary: revision.Summary, Status: revision.Status,
		SmallTask: revision.SmallTask, TaskID: revision.TaskID,
	}, true, nil
}
