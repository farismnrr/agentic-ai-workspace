package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// CreateInput is what the composer sends.
type CreateInput struct {
	// Team says whether the agent leads a team; nil uses the person's choice for new tasks.
	Team *bool
	// PlanFirst says whether the team stops at the PRD and waits for a yes. It is not offered to the person: nil, the
	// default, always waits. Only code that starts a task on someone's behalf may turn it off.
	PlanFirst  *bool
	ProjectID  string
	Prompt     string
	ProviderID string
	Model      string
	// Effort is how hard the model should think; empty uses the person's default.
	Effort domain.Effort
}

// Create starts a task: it records the request and queues the agent.
func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (domain.Task, error) {
	prompt := strings.TrimSpace(in.Prompt)
	if err := validatePrompt(prompt); err != nil {
		return domain.Task{}, err
	}
	if _, err := s.deps.Projects.Name(ctx, userID, in.ProjectID); err != nil {
		return domain.Task{}, domain.ErrNoProject
	}

	followSettings := in.followsSettings()
	in, team := s.route(ctx, userID, in)
	providerID, model, err := s.chooseModel(ctx, userID, in)
	if err != nil {
		return domain.Task{}, err
	}
	effort, err := s.chooseEffort(ctx, userID, in.Effort)
	if err != nil {
		return domain.Task{}, err
	}
	id, err := s.deps.IDs.ID()
	if err != nil {
		return domain.Task{}, fmt.Errorf("new task id: %w", err)
	}

	now := s.deps.Now()
	task := domain.Task{
		ID: id, ProjectID: in.ProjectID, UserID: userID, Title: titleOf(prompt), ProviderID: providerID,
		FollowSettings: followSettings, Model: model, Effort: effort, Team: team,
		PlanFirst: in.PlanFirst == nil || *in.PlanFirst,
		Status:    domain.StatusQueued, CreatedAt: now, UpdatedAt: now,
		Plan: []domain.PlanStep{},
	}
	if err := s.deps.Tasks.Create(ctx, task); err != nil {
		return domain.Task{}, fmt.Errorf("save task: %w", err)
	}
	if err := s.say(ctx, task.ID, in.Prompt); err != nil {
		return domain.Task{}, err
	}
	s.dispatcher.Enqueue(task.ID, task.ProjectID)
	return s.withWindow(ctx, task), nil
}
