package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const orchestratorListLimit = 100

// ListOrchestratorProjects returns the caller's active projects for MCP selection.
func (s *Service) ListOrchestratorProjects(ctx context.Context, userID string) ([]ProjectSummary, error) {
	projects, err := s.deps.OrchestratorProjects.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list Orchestrator projects: %w", err)
	}
	if len(projects) > orchestratorListLimit {
		projects = projects[:orchestratorListLimit]
	}
	return projects, nil
}

// ListOrchestratorTasks returns active team tasks in an owned project, newest first.
func (s *Service) ListOrchestratorTasks(
	ctx context.Context, userID, projectID string,
) ([]domain.Task, error) {
	if userID == "" {
		return nil, domain.ErrNotFound
	}
	if _, err := s.deps.Projects.Name(ctx, userID, projectID); err != nil {
		return nil, domain.ErrNoProject
	}
	tasks, err := s.deps.OrchestratorTasks.ListOrchestrator(ctx, userID, projectID, orchestratorListLimit)
	if err != nil {
		return nil, fmt.Errorf("list Orchestrator tasks: %w", err)
	}
	return tasks, nil
}

// StartOrchestrator creates and queues a task that must run with the project team.
func (s *Service) StartOrchestrator(ctx context.Context, userID, projectID, prompt string) (domain.Task, error) {
	team := true

	return s.Create(ctx, userID, CreateInput{ProjectID: projectID, Prompt: prompt, Team: &team})
}

// GetOrchestrator returns a task only when it belongs to this owner and project and uses Orchestrator.
func (s *Service) GetOrchestrator(ctx context.Context, userID, projectID, taskID string) (domain.Task, error) {
	task, err := s.orchestratorTask(ctx, userID, projectID, taskID)
	if err != nil {
		return domain.Task{}, err
	}

	return s.withWindow(ctx, task), nil
}

// SendOrchestratorMessage follows up on an eligible Orchestrator task using the existing task rules.
func (s *Service) SendOrchestratorMessage(
	ctx context.Context, userID, projectID, taskID, text string,
) error {
	task, err := s.orchestratorTask(ctx, userID, projectID, taskID)
	if err != nil {
		return err
	}
	if task.Status == domain.StatusReview {
		return domain.ErrWrongState
	}

	return s.FollowUp(ctx, userID, taskID, text)
}

// RetryOrchestrator requeues only failed Orchestrator tasks through the existing retry rules.
func (s *Service) RetryOrchestrator(ctx context.Context, userID, projectID, taskID string) error {
	if _, err := s.orchestratorTask(ctx, userID, projectID, taskID); err != nil {
		return err
	}

	return s.Retry(ctx, userID, taskID)
}

func (s *Service) orchestratorTask(ctx context.Context, userID, projectID, taskID string) (domain.Task, error) {
	if userID == "" {
		return domain.Task{}, domain.ErrNotFound
	}

	if _, err := s.deps.Projects.Name(ctx, userID, projectID); err != nil {
		return domain.Task{}, domain.ErrNoProject
	}

	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if task.ProjectID != projectID {
		return domain.Task{}, domain.ErrNotFound
	}
	if !task.Team || task.Source != "" {
		return domain.Task{}, domain.ErrWrongState
	}

	return task, nil
}
