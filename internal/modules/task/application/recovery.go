package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

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
		if err := s.recoverTask(ctx, task); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (s *Service) recoverTask(ctx context.Context, task domain.Task) error {
	// External Orchestrator tasks own their worker jobs. Never hand them to the
	// internal lead or silently mark a still-active external issue completed.
	if strings.HasPrefix(task.Source, externalLeadSource) {
		return s.recoverExternalLeadTask(ctx, task)
	}
	if task.WorkID != "" {
		return nil
	}
	if task.OriginTaskID != "" && s.handoffs != nil {
		recovering, err := s.handoffs.RecoveringTask(ctx, task.ID)
		if err != nil || recovering {
			return err
		}
	}
	if task.Source != "" {
		if err := s.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusDone, s.deps.Now()); err != nil {
			return fmt.Errorf("finish recovered external task: %w", err)
		}
		return nil
	}
	return s.fail(ctx, task.ID, "The app restarted while I was working.", "Press Continue to pick up where I left off.")
}

func (s *Service) recoverExternalLeadTask(ctx context.Context, task domain.Task) error {
	events, err := s.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return fmt.Errorf("read interrupted external task: %w", err)
	}
	approval, err := s.ExternalApproval(ctx, task.UserID, task.ID)
	if err != nil {
		return fmt.Errorf("check external plan approval: %w", err)
	}
	if unresolvedQuestion(events) || (approval.Status != "approved" && approval.Status != "not-required-yet") {
		return s.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusNeedsInput, s.now())
	}
	return s.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusWorking, s.now())
}

func unresolvedQuestion(events []domain.Event) bool {
	for _, event := range events {
		if event.Kind == domain.EventQuestion && event.Answered == "" {
			return true
		}
	}
	return false
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
