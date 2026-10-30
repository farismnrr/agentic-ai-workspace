package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func planSnapshot(work domain.ExternalWork, task domain.Task, in SubmitPlanInput, summary, qa string) (
	domain.PlanRevision, error,
) {
	if len(task.Plan) == 0 {
		return domain.PlanRevision{}, fmt.Errorf("plan steps are required: %w", domain.ErrBadRequest)
	}
	if !in.SmallTask && strings.TrimSpace(task.PRD) == "" {
		return domain.PlanRevision{}, fmt.Errorf(
			"requirements document (save_requirements) is required before submitting full plan: %w",
			domain.ErrBadRequest,
		)
	}
	revision := domain.PlanRevision{
		WorkID: work.ID, TaskID: task.ID, PlanRev: work.PlanRev, Steps: task.Plan,
		PRD: task.PRD, Research: task.Research, QA: qa, Summary: summary, SmallTask: in.SmallTask,
		Status: domain.PlanRevisionSubmitted,
	}
	digest, err := planDigest(revision)
	if err != nil {
		return domain.PlanRevision{}, err
	}
	revision.Digest = digest
	return revision, nil
}

func (w *Works) requireNoOpenQuestions(ctx context.Context, workID string) error {
	if w.questions == nil {
		return nil
	}
	open, err := w.questions.OpenQuestionCount(ctx, workID)
	if err != nil {
		return fmt.Errorf("count open work questions: %w", err)
	}
	if open > 0 {
		return domain.ErrOpenQuestions
	}
	return nil
}

func (w *Works) requireResearch(ctx context.Context, work domain.ExternalWork, task domain.Task,
	smallTask bool,
) error {
	if smallTask {
		return nil
	}
	events, err := w.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return fmt.Errorf("read research history: %w", err)
	}
	if enough, _ := domain.ResearchEnough(researchRounds(events, work.ID)); !enough {
		return domain.ErrNoResearch
	}
	return nil
}

func researchRounds(events []domain.Event, workID string) []string {
	rounds := []string{}
	for _, event := range events {
		if event.Kind == domain.EventResearch && event.Work != nil && event.Work.ID == workID {
			rounds = append(rounds, event.Technical)
		}
	}
	return rounds
}

func (w *Works) submitPreparedPlan(ctx context.Context, work domain.ExternalWork,
	revision domain.PlanRevision,
) (SubmitPlanResult, error) {
	prior, existing, err := w.existingSubmission(ctx, work, revision)
	if err != nil {
		return SubmitPlanResult{}, err
	}
	if existing {
		return prior, nil
	}
	if work.PlanState != domain.PlanDraft || !work.PlanState.CanMoveTo(domain.PlanReadyForApproval) {
		return SubmitPlanResult{}, domain.ErrWrongState
	}
	revision.ID, err = w.deps.IDs.ID()
	if err != nil {
		return SubmitPlanResult{}, fmt.Errorf("new plan revision id: %w", err)
	}
	event := workEvent(work, domain.EventPlanReview, "Plan ready for approval", "", w.now())
	event.PlanReview = &domain.PlanReview{
		Summary: revision.Summary, Steps: revision.Steps, Status: domain.PlanRevisionSubmitted,
	}
	result, err := w.plans.SubmitPlan(ctx, PlanSubmission{
		WorkID: work.ID, UserID: work.UserID, ExpectRev: work.Rev, Revision: revision, Event: event,
	})
	if err != nil {
		if result, recovered := w.recoverSubmittedPlan(ctx, work, revision, err); recovered {
			return result, nil
		}
		return SubmitPlanResult{}, fmt.Errorf("submit plan snapshot: %w", err)
	}
	w.notify(ctx, domain.Task{ID: work.TaskID, UserID: work.UserID})
	return SubmitPlanResult{Revision: result, WorkRev: work.Rev + 1, WorkState: work.State}, nil
}

func (w *Works) existingSubmission(ctx context.Context, work domain.ExternalWork, revision domain.PlanRevision) (
	SubmitPlanResult, bool, error,
) {
	prior, found, err := w.plans.LatestPlan(ctx, work.ID)
	if err != nil {
		return SubmitPlanResult{}, false, fmt.Errorf("read plan revision: %w", err)
	}
	exists := found && work.PlanState == domain.PlanReadyForApproval && prior.Digest == revision.Digest &&
		prior.Status == domain.PlanRevisionSubmitted
	if !exists {
		return SubmitPlanResult{}, false, nil
	}
	return SubmitPlanResult{Revision: prior, WorkRev: work.Rev, WorkState: work.State}, true, nil
}

func (w *Works) recoverSubmittedPlan(ctx context.Context, work domain.ExternalWork,
	revision domain.PlanRevision, submitErr error,
) (SubmitPlanResult, bool) {
	if !errors.Is(submitErr, domain.ErrStaleRevision) {
		return SubmitPlanResult{}, false
	}
	latest, exists, err := w.plans.LatestPlan(ctx, work.ID)
	if err != nil || !exists || latest.Digest != revision.Digest || latest.Status != domain.PlanRevisionSubmitted {
		return SubmitPlanResult{}, false
	}
	return SubmitPlanResult{Revision: latest, WorkRev: work.Rev + 1, WorkState: work.State}, true
}

func planDigest(revision domain.PlanRevision) (string, error) {
	//nolint:govet // fieldalignment: field order is part of the persisted revision digest.
	data, err := json.Marshal(struct {
		PlanRev  int64
		Steps    []domain.PlanStep
		PRD      string
		Research string
		QA       string
		Summary  string
		Small    bool
	}{
		PlanRev: revision.PlanRev, Steps: revision.Steps, PRD: revision.PRD,
		Research: revision.Research, QA: revision.QA,
		Summary: revision.Summary, Small: revision.SmallTask,
	})
	if err != nil {
		return "", fmt.Errorf("encode plan revision: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
