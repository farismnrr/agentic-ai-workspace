package application

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// say records something the person wrote, in the thread and for the model.
func (s *Service) say(ctx context.Context, taskID, text string) error {
	event := domain.Event{Kind: domain.EventRequest, Text: text, Time: s.deps.Now()}
	if _, err := s.deps.Events.AppendEvent(ctx, taskID, event); err != nil {
		return fmt.Errorf("record request: %w", err)
	}

	block := domain.Block{Kind: domain.BlockText, Text: text}
	if err := s.deps.Transcript.AppendBlock(ctx, taskID, domain.RoleUser, block); err != nil {
		return fmt.Errorf("record message: %w", err)
	}

	return nil
}

func (s *Service) requeue(ctx context.Context, task domain.Task) error {
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

func modeOf(mode domain.Mode) domain.Mode {
	if mode == domain.ModeAlways {
		return domain.ModeAlways
	}

	return domain.ModeRisky
}

func titleOf(prompt string) string {
	line, _, _ := strings.Cut(prompt, "\n")
	if utf8.RuneCountInString(line) <= maxTitle {
		return line
	}

	return string([]rune(line)[:maxTitle]) + messageCuts
}
