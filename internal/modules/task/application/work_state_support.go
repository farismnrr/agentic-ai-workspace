package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (w *Works) readWorkSnapshot(ctx context.Context, userID, client, projectID, workID string) (
	domain.ExternalWork, domain.Task, []domain.Event, error,
) {
	work, err := w.store.FindWork(ctx, userID, client, projectID, workID)
	if err != nil {
		return domain.ExternalWork{}, domain.Task{}, nil, domain.ErrUnknownWork
	}
	task, err := w.deps.Tasks.Find(ctx, userID, work.TaskID)
	if err != nil || task.ProjectID != projectID || task.Source != client || task.WorkID != work.ID {
		return domain.ExternalWork{}, domain.Task{}, nil, domain.ErrUnknownWork
	}
	events, err := w.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return domain.ExternalWork{}, domain.Task{}, nil, fmt.Errorf("read work events: %w", err)
	}
	return work, task, events, nil
}

func (w *Works) latestReview(ctx context.Context, workID string) (*domain.PlanReview, error) {
	review, found, err := w.plans.LatestPlanReview(ctx, workID)
	if err != nil {
		return nil, fmt.Errorf("read latest plan revision: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &review, nil
}

func (w *Works) questionGroups(ctx context.Context, workID string) (QuestionGroups, error) {
	questions, err := w.Questions(ctx, workID)
	if err != nil {
		return QuestionGroups{}, err
	}
	groups := QuestionGroups{Open: []WorkQuestion{}, Answered: []WorkQuestion{}}
	for _, question := range questions {
		switch question.State {
		case "open":
			groups.Open = append(groups.Open, question)
		case "answered":
			groups.Answered = append(groups.Answered, question)
		}
	}
	return groups, nil
}

func stateNextAction(state StateResult, events []domain.Event, plan []domain.PlanStep) string {
	if len(state.Questions.Open) > 0 {
		return "wait_for_answers"
	}
	if state.PlanState == domain.PlanReadyForApproval {
		return "wait_for_approval"
	}
	if len(plan) == 0 {
		return "save_plan"
	}
	if ready, _ := domain.ResearchEnough(researchRounds(events, state.WorkID)); !ready {
		return "finish_research"
	}
	return "submit_plan"
}

func stateSnapshot(work domain.ExternalWork, task domain.Task, events []domain.Event, now time.Time,
	thresholds domain.WorkPhaseThresholds,
) StateResult {
	state := StateResult{
		TaskID: work.TaskID, WorkID: work.ID, Revision: work.Rev, State: work.State, PlanState: work.PlanState,
		Questions: QuestionGroups{Open: []WorkQuestion{}, Answered: []WorkQuestion{}}, Role: canonicalMemberKey(work.RoleKey),
		Plan: task.Plan, PlanRevision: work.PlanRev, ResearchSize: len(task.Research), Research: preview(task.Research),
		PRDSize: len(task.PRD), PRD: preview(task.PRD), Progress: []string{},
		Stale: domain.DeriveWorkPhase(work, now, thresholds) == domain.WorkPhaseIdle,
	}
	for _, event := range events {
		countEvidence(&state, event)
		if progress, ok := progressLine(work.ID, event); ok {
			state.Progress = append(state.Progress, progress)
		}
	}
	if len(state.Progress) > workProgressLimit {
		state.Progress = state.Progress[len(state.Progress)-workProgressLimit:]
	}
	return boundedState(state)
}

func countEvidence(state *StateResult, event domain.Event) {
	if event.Work == nil {
		return
	}
	switch event.Work.Evidence {
	case domain.EvidenceClaimed:
		state.Evidence.Claimed++
	case domain.EvidenceObserved:
		state.Evidence.Observed++
	case domain.EvidenceRan:
		state.Evidence.Ran++
	}
}

func progressLine(workID string, event domain.Event) (string, bool) {
	if event.Work == nil || event.Work.ID != workID || event.Kind == domain.EventRequest ||
		strings.TrimSpace(event.Text) == "" {
		return "", false
	}
	return truncateRunes(event.Text, workProgressPreview), true
}

func preview(text string) string { return truncateRunes(text, workDocumentPreview) }

func truncateRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "…"
}

func boundedState(state StateResult) StateResult {
	for len(mustJSON(state)) > workStateMaxBytes {
		if len(state.Questions.Answered) > 0 {
			state.Questions.Answered = state.Questions.Answered[1:]
			continue
		}
		if len(state.Questions.Open) > 1 {
			state.Questions.Open = state.Questions.Open[1:]
			continue
		}
		if len(state.Progress) > 0 {
			state.Progress = state.Progress[1:]
			continue
		}
		if len(state.Plan) > 0 {
			state.Plan = state.Plan[:len(state.Plan)-1]
			continue
		}
		state.Research = truncateRunes(state.Research, workMinimalPreview)
		state.PRD = truncateRunes(state.PRD, workMinimalPreview)
		break
	}
	return state
}

func mustJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}
