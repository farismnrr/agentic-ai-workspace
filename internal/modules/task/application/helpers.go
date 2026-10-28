package application

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (s *Service) requeue(ctx context.Context, task domain.Task) error {
	if task.Source != "" {
		return domain.ErrWrongState
	}
	if err := s.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusQueued, s.deps.Now()); err != nil {
		return fmt.Errorf("queue task: %w", err)
	}

	s.dispatcher.Enqueue(task.ID, task.ProjectID)

	return nil
}

// owned loads a task the person owns.
func (s *Service) owned(ctx context.Context, userID, taskID string) (domain.Task, error) {
	task, err := s.deps.Tasks.Find(ctx, userID, taskID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("find task: %w", err)
	}

	return task, nil
}

func titleOf(prompt string) string {
	line, _, _ := strings.Cut(prompt, "\n")
	if utf8.RuneCountInString(line) <= maxTitle {
		return line
	}

	return string([]rune(line)[:maxTitle]) + messageCuts
}

func validatePrompt(text string) error {
	if text == "" {
		return domain.ErrBadRequest
	}
	return nil
}
