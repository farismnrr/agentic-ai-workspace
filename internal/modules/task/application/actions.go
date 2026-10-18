package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Answer is the person's reply to a question or approval.
type Answer struct {
	Allow *bool
	// Choice answers a question with one reply; Answers holds one list of replies per question of a form.
	Choice  string
	Answers [][]string
}

// Answer resolves a pending approval or question and lets the agent continue.
func (s *Service) Answer(ctx context.Context, userID, taskID, eventID string, answer Answer) error {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}

	event, err := s.deps.Events.FindEvent(ctx, taskID, eventID)
	if err != nil {
		return fmt.Errorf("find event: %w", err)
	}

	if err := resolve(&event, answer); err != nil {
		return err
	}

	if err := s.deps.Events.UpdateEvent(ctx, taskID, event); err != nil {
		return fmt.Errorf("save answer: %w", err)
	}

	return s.requeue(ctx, task)
}

func resolve(event *domain.Event, answer Answer) error {
	switch {
	case event.Kind == domain.EventApproval && event.Resolved == "" && answer.Allow != nil:
		event.Resolved = domain.Denied
		if *answer.Allow {
			event.Resolved = domain.Allowed
		}
	case event.Kind == domain.EventQuestion && event.Answered == "":
		summary, err := answerQuestion(*event, answer)
		if err != nil {
			return err
		}

		event.Answered = summary
	default:
		return domain.ErrNotWaiting
	}

	return nil
}

// Cancel stops a task that is working or queued.
func (s *Service) Cancel(ctx context.Context, userID, taskID string) error {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}

	switch s.dispatcher.Cancel(task.ID) {
	case CancelRunning:
		return nil
	case CancelQueued:
		return s.fail(ctx, task.ID, "Stopped before I started.", "Press Continue when you want me to begin.")
	case CancelNone:
	}

	return domain.ErrWrongState
}

// Accept marks reviewed work as done.
func (s *Service) Accept(ctx context.Context, userID, taskID string) error {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}

	if task.Status != domain.StatusReview {
		return domain.ErrWrongState
	}

	if err := s.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusDone, s.deps.Now()); err != nil {
		return fmt.Errorf("accept task: %w", err)
	}

	return nil
}

// Retry continues a stopped or failed task from where it left off.
func (s *Service) Retry(ctx context.Context, userID, taskID string) error {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}

	if task.Status != domain.StatusFailed {
		return domain.ErrWrongState
	}

	return s.requeue(ctx, task)
}

// UndoChange puts back what an earlier step changed, unless it changed again since.
func (s *Service) UndoChange(ctx context.Context, userID, taskID, eventID string) error {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return err
	}

	event, err := s.deps.Events.FindEvent(ctx, taskID, eventID)
	if err != nil {
		return fmt.Errorf("find event: %w", err)
	}

	if !event.CanUndo() {
		return domain.ErrWrongState
	}

	if task.Status.Running() {
		return domain.ErrBusy
	}

	if err := s.deps.Tools.Undo(ctx, task.ProjectID, event.Undo); err != nil {
		return fmt.Errorf("undo: %w", err)
	}

	event.Undone = true

	if err := s.deps.Events.UpdateEvent(ctx, taskID, event); err != nil {
		return fmt.Errorf("record undo: %w", err)
	}

	return nil
}

// Usage sums the person's model usage since a date.
func (s *Service) Usage(ctx context.Context, userID string, since time.Time) ([]UsageRow, error) {
	rows, err := s.deps.Usage.UsageSummary(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("summarise usage: %w", err)
	}

	return rows, nil
}

// Recover marks tasks that were running when the app stopped, so nothing is stuck.
func (s *Service) Recover(ctx context.Context) error {
	tasks, err := s.deps.Tasks.Running(ctx)
	if err != nil {
		return fmt.Errorf("find interrupted tasks: %w", err)
	}

	var failures []error

	for _, task := range tasks {
		err := s.fail(ctx, task.ID, "The app restarted while I was working.", "Press Continue to pick up where I left off.")
		if err != nil {
			failures = append(failures, err)
		}
	}

	return errors.Join(failures...)
}

func (s *Service) fail(ctx context.Context, taskID, text, hint string) error {
	if _, err := s.deps.Events.AppendEvent(ctx, taskID, domain.Event{
		Kind: domain.EventError, Text: text, Hint: hint, Time: s.deps.Now(),
	}); err != nil {
		return fmt.Errorf("record failure: %w", err)
	}

	if err := s.deps.Tasks.SetStatus(ctx, taskID, domain.StatusFailed, s.deps.Now()); err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}

	return nil
}
