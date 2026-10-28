package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	documentThreshold    = 12_000
	maxTitle             = 60
	listLimit            = 200
	messageCuts          = "…"
	defaultWorkIdleAfter = 30 * time.Minute
)

// Service holds the use cases people trigger from the app.
type Service struct {
	deps              Deps
	dispatcher        Dispatcher
	workStopper       WorkStopper
	workSummaries     WorkSummaries
	externalQuestions QuestionAnswerStore
	planReviews       *PlanReviews
	handoffs          *Handoffs
	planSnapshots     PlanSnapshotReader
	// calls counts the calls of each connected app's thread that are under way, so the thread shows as working
	// until the last one ends.
	calls map[string]int
	// recording keeps two calls from a connected app from starting two threads.
	recording     sync.Mutex
	workIdleAfter time.Duration
}

// NewService wires the use cases to the same collaborators as the runner.
func NewService(deps Deps, dispatcher Dispatcher) *Service {
	return &Service{
		deps: deps, dispatcher: dispatcher, calls: map[string]int{},
		workIdleAfter: defaultWorkIdleAfter,
	}
}

// UseWorkIdleAfter sets the idle threshold used in external-work summaries.
func (s *Service) UseWorkIdleAfter(after time.Duration) {
	if after > 0 {
		s.workIdleAfter = after
	}
}

func (s *Service) now() time.Time {
	if s.deps.Now != nil {
		return s.deps.Now()
	}
	return time.Now()
}

// FollowUp adds a message to a task that is waiting for review, finished or stopped.
func (s *Service) FollowUp(ctx context.Context, userID, taskID, text string) error {
	if err := validatePrompt(strings.TrimSpace(text)); err != nil {
		return err
	}

	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if task.ArchivedAt != nil {
		return domain.ErrWrongState
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

// Pin records the first time the person pinned their task.
func (s *Service) Pin(ctx context.Context, userID, taskID string) (domain.Task, error) {
	current, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if current.ArchivedAt != nil {
		return domain.Task{}, domain.ErrWrongState
	}
	task, err := s.deps.Tasks.Pin(ctx, userID, taskID, s.deps.Now())
	if err != nil {
		return domain.Task{}, fmt.Errorf("pin task: %w", err)
	}

	return s.withWindow(ctx, task), nil
}

// Unpin clears the person's pin from their task.
func (s *Service) Unpin(ctx context.Context, userID, taskID string) (domain.Task, error) {
	current, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if current.ArchivedAt != nil {
		return domain.Task{}, domain.ErrWrongState
	}
	task, err := s.deps.Tasks.Unpin(ctx, userID, taskID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("unpin task: %w", err)
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
