package application

import (
	"context"
	"errors"
	"fmt"
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
		if task.WorkID != "" {
			continue
		}
		if task.Source != "" {
			if err := s.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusDone, s.deps.Now()); err != nil {
				failures = append(failures, err)
			}
			continue
		}
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
