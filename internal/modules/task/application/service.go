package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	maxPrompt   = 4000
	maxTitle    = 60
	listLimit   = 200
	messageCuts = "…"
)

// Service holds the use cases people trigger from the app.
type Service struct {
	deps       Deps
	dispatcher Dispatcher
	// recording keeps two calls from a connected app from starting two threads.
	recording sync.Mutex
}

// NewService wires the use cases to the same collaborators as the runner.
func NewService(deps Deps, dispatcher Dispatcher) *Service {
	return &Service{deps: deps, dispatcher: dispatcher}
}

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
	if prompt == "" || utf8.RuneCountInString(prompt) > maxPrompt {
		return domain.Task{}, domain.ErrBadRequest
	}

	if _, err := s.deps.Projects.Name(ctx, userID, in.ProjectID); err != nil {
		return domain.Task{}, domain.ErrNoProject
	}

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
		Model: model, Effort: effort, Team: team, PlanFirst: in.PlanFirst == nil || *in.PlanFirst,
		Status: domain.StatusQueued, CreatedAt: now, UpdatedAt: now,
		Plan: []domain.PlanStep{},
	}

	if err := s.deps.Tasks.Create(ctx, task); err != nil {
		return domain.Task{}, fmt.Errorf("save task: %w", err)
	}

	if err := s.say(ctx, task.ID, prompt); err != nil {
		return domain.Task{}, err
	}

	s.dispatcher.Enqueue(task.ID, task.ProjectID)

	return task, nil
}

// FollowUp adds a message to a task that is waiting for review, finished or stopped.
func (s *Service) FollowUp(ctx context.Context, userID, taskID, text string) error {
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > maxPrompt {
		return domain.ErrBadRequest
	}

	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}

	switch {
	case task.Source != "":
		return domain.ErrWrongState
	case task.Status.Running():
		return domain.ErrBusy
	case task.Status == domain.StatusNeedsInput:
		return domain.ErrNeedsAnswer
	}

	if err := s.say(ctx, task.ID, text); err != nil {
		return err
	}

	return s.requeue(ctx, task)
}

// Get returns one of the person's tasks.
func (s *Service) Get(ctx context.Context, userID, taskID string) (domain.Task, error) {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return domain.Task{}, err
	}

	return s.withWindow(ctx, task), nil
}

// List returns the person's tasks, newest first; projectID may be empty.
func (s *Service) List(ctx context.Context, userID, projectID string) ([]domain.Task, error) {
	tasks, err := s.deps.Tasks.List(ctx, userID, projectID, listLimit)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	for i := range tasks {
		tasks[i] = s.withWindow(ctx, tasks[i])
	}

	return tasks, nil
}

// Events returns the thread entries after a cursor.
func (s *Service) Events(ctx context.Context, userID, taskID string, after int64) ([]domain.Event, error) {
	if _, err := s.owned(ctx, userID, taskID); err != nil {
		return nil, err
	}

	events, err := s.deps.Events.EventsAfter(ctx, taskID, after)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	return events, nil
}
