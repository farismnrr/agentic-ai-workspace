package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// buildLinkedTask shares routing and model selection for external chat and approved plan handoffs.
func (s *Service) buildLinkedTask(ctx context.Context, external domain.Task, message string) (
	domain.Task, domain.Event, error,
) {
	input := CreateInput{ProjectID: external.ProjectID}
	input, team := s.route(ctx, external.UserID, input)
	providerID, model, err := s.chooseModel(ctx, external.UserID, input)
	if err != nil {
		return domain.Task{}, domain.Event{}, err
	}
	effort, err := s.chooseEffort(ctx, external.UserID, domain.EffortDefault)
	if err != nil {
		return domain.Task{}, domain.Event{}, err
	}
	id, err := s.deps.IDs.ID()
	if err != nil {
		return domain.Task{}, domain.Event{}, fmt.Errorf("new linked task id: %w", err)
	}
	now := s.deps.Now()
	task := domain.Task{
		ID: id, ProjectID: external.ProjectID, UserID: external.UserID, Title: titleOf(message),
		ProviderID: providerID, Model: model, Effort: effort, Team: team, PlanFirst: false,
		Status: domain.StatusQueued, CreatedAt: now, UpdatedAt: now, Plan: []domain.PlanStep{},
		OriginTaskID: external.ID,
	}
	event := domain.Event{Kind: domain.EventRequest, Text: message, Time: now}
	return task, event, nil
}
