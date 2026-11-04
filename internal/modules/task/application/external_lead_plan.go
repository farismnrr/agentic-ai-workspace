package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type ExternalPlanApproval struct {
	Status  string `json:"status"`
	EventID string `json:"event_id,omitempty"`
}

// UpdateExternalPlan records the external lead's todo list in the primary task.
// This is not an internal lead turn and does not dispatch a model.
func (s *Service) UpdateExternalPlan(ctx context.Context, userID, taskID string, steps []domain.PlanStep) error {
	task, err := s.externalOwned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if len(steps) == 0 || len(steps) > 40 {
		return domain.ErrBadRequest
	}
	ids := make(map[string]bool, len(steps))
	var lines []string
	for _, step := range steps {
		if step.ID == "" || strings.TrimSpace(step.Title) == "" || ids[step.ID] {
			return domain.ErrBadRequest
		}
		if step.State != domain.PlanPending && step.State != domain.PlanActive &&
			step.State != domain.PlanDone && step.State != domain.PlanFailed {
			return domain.ErrBadRequest
		}
		ids[step.ID] = true
		lines = append(lines, fmt.Sprintf("[%s] %s", step.State, step.Title))
	}
	if err := s.deps.Tasks.SavePlan(ctx, task.ID, steps, s.now()); err != nil {
		return err
	}
	isProgress := len(task.Plan) > 0 && len(task.Plan) == len(steps)
	if isProgress {
		for i := range task.Plan {
			if task.Plan[i].ID != steps[i].ID || task.Plan[i].Title != steps[i].Title {
				isProgress = false
				break
			}
		}
	}
	text := "External orchestrator updated the plan"
	if isProgress {
		text = "External orchestrator updated step progress"
	}
	_, err = s.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventPlan, Text: text,
		Technical: strings.Join(lines, "\n"), Agent: task.Source, Time: s.now(),
	})
	return err
}

func (s *Service) SubmitExternalPlan(ctx context.Context, userID, taskID string) (ExternalPlanApproval, error) {
	task, err := s.externalOwned(ctx, userID, taskID)
	if err != nil {
		return ExternalPlanApproval{}, err
	}
	events, err := s.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return ExternalPlanApproval{}, err
	}
	prd := lastSeq(events, domain.EventPRD)
	if prd == 0 || len(task.Plan) == 0 {
		return ExternalPlanApproval{}, domain.ErrWrongState
	}
	prd = max(prd, lastPlanRevisionSeq(events))
	if approvedAfter(events, task.Source, prd) {
		return ExternalPlanApproval{Status: "approved"}, nil
	}
	for _, event := range events {
		if event.Seq > prd && event.Kind == domain.EventQuestion && event.Agent == task.Source &&
			event.Answered == "" {
			return ExternalPlanApproval{Status: "pending", EventID: event.ID}, nil
		}
	}
	event, err := s.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventQuestion, Agent: task.Source, Time: s.now(),
		Text: "The external orchestrator submitted the PRD. Start building?",
		Questions: []domain.Question{{
			Text: "Approve the PRD and start implementation?", Type: domain.QuestionSingle,
			Options: []domain.Option{
				{Label: startBuilding, Description: "Allow internal workers to implement this PRD"},
				{Label: "Change something", Description: "Revise the PRD before implementation"},
			},
		}},
	})
	if err != nil {
		return ExternalPlanApproval{}, err
	}
	if err := s.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusNeedsInput, s.now()); err != nil {
		return ExternalPlanApproval{}, err
	}
	return ExternalPlanApproval{Status: "pending", EventID: event.ID}, nil
}

func (s *Service) ExternalApproval(ctx context.Context, userID, taskID string) (ExternalPlanApproval, error) {
	task, err := s.externalOwned(ctx, userID, taskID)
	if err != nil {
		return ExternalPlanApproval{}, err
	}
	events, err := s.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return ExternalPlanApproval{}, err
	}
	prd := lastSeq(events, domain.EventPRD)
	if prd == 0 {
		return ExternalPlanApproval{Status: "not-required-yet"}, nil
	}
	prd = max(prd, lastPlanRevisionSeq(events))
	if approvedAfter(events, task.Source, prd) {
		return ExternalPlanApproval{Status: "approved"}, nil
	}
	for _, event := range events {
		if event.Seq > prd && event.Kind == domain.EventQuestion && event.Agent == task.Source {
			if event.Answered == "" {
				return ExternalPlanApproval{Status: "pending", EventID: event.ID}, nil
			}
			return ExternalPlanApproval{Status: "revision-requested", EventID: event.ID}, nil
		}
	}
	return ExternalPlanApproval{Status: "not-submitted"}, nil
}
