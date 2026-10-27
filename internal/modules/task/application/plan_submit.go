package application

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// SubmitPlanInput is the explicit request to make a saved plan ready for a person's review.
type SubmitPlanInput struct {
	Summary      string
	RequestID    string
	BaseRevision int64
	SmallTask    bool
}

// SubmitPlanResult identifies the immutable revision awaiting review.
type SubmitPlanResult struct {
	WorkState domain.WorkState    `json:"work_state"`
	Revision  domain.PlanRevision `json:"revision"`
	WorkRev   int64               `json:"work_revision"`
}

// SubmitPlan snapshots saved plan material and moves the work into review.
func (w *Works) SubmitPlan(ctx context.Context, userID, client, projectID, workID string,
	in SubmitPlanInput,
) (SubmitPlanResult, error) {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return SubmitPlanResult{}, fmt.Errorf("submit plan snapshot: %w", err)
	}
	if !validKey(in.RequestID) || utf8.RuneCountInString(in.Summary) > domain.MaxWorkSummary {
		return SubmitPlanResult{}, domain.ErrBadRequest
	}
	work, revision, err := w.preparePlanSubmission(ctx, userID, client, projectID, workID, in)
	if err != nil {
		return SubmitPlanResult{}, err
	}
	return w.submitPreparedPlan(ctx, work, revision)
}

func (w *Works) preparePlanSubmission(ctx context.Context, userID, client, projectID, workID string,
	in SubmitPlanInput,
) (domain.ExternalWork, domain.PlanRevision, error) {
	summary, err := sanitizeText(in.Summary, domain.MaxWorkSummary)
	if err != nil || summary == "" {
		return domain.ExternalWork{}, domain.PlanRevision{}, domain.ErrBadRequest
	}
	work, task, _, err := w.mutable(ctx, userID, client, projectID, workID)
	if err != nil {
		return domain.ExternalWork{}, domain.PlanRevision{}, err
	}
	if in.BaseRevision != work.PlanRev {
		return domain.ExternalWork{}, domain.PlanRevision{}, domain.ErrStaleRevision
	}
	if questionErr := w.requireNoOpenQuestions(ctx, work.ID); questionErr != nil {
		return domain.ExternalWork{}, domain.PlanRevision{}, questionErr
	}
	if researchErr := w.requireResearch(ctx, work, task, in.SmallTask); researchErr != nil {
		return domain.ExternalWork{}, domain.PlanRevision{}, researchErr
	}
	qa, err := w.qaSnapshot(ctx, work.ID)
	if err != nil {
		return domain.ExternalWork{}, domain.PlanRevision{}, err
	}
	revision, err := planSnapshot(work, task, in, summary, qa)
	if err != nil {
		return domain.ExternalWork{}, domain.PlanRevision{}, err
	}
	return work, revision, nil
}
