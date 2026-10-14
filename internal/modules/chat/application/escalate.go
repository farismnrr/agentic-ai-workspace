package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/domain"
)

var errNoTaskCreator = errors.New("task creator is not wired")

const maxContextMessages = 6

// EscalateInput specifies parameters for turning chat into a task.
type EscalateInput struct {
	ProjectID string
	Prompt    string
	Mode      string
}

// Escalate converts the current conversation intent into a new formal task.
func (s *Service) Escalate(ctx context.Context, userID string, in EscalateInput) (string, error) {
	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" {
		chat, msgs, err := s.Conversation(ctx, userID, in.ProjectID)
		if err != nil {
			return "", err
		}
		prompt = buildEscalationPrompt(chat.Title, msgs)
	}

	if s.deps.TaskCreator == nil {
		return "", errNoTaskCreator
	}

	mode := in.Mode
	if mode == "" {
		mode = "risky"
	}

	taskID, err := s.deps.TaskCreator.CreateTask(ctx, userID, TaskInput{
		ProjectID: in.ProjectID,
		Prompt:    prompt,
		Mode:      mode,
	})
	if err != nil {
		return "", fmt.Errorf("create task from chat: %w", err)
	}

	return taskID, nil
}

func buildEscalationPrompt(title string, msgs []domain.Message) string {
	if len(msgs) == 0 {
		return title
	}

	var sb strings.Builder
	sb.WriteString("Task escalated from project chat discussion:\n\n")

	start := 0
	if len(msgs) > maxContextMessages {
		start = len(msgs) - maxContextMessages
	}

	for _, m := range msgs[start:] {
		roleName := "User"
		if m.Role == domain.RoleAssistant {
			roleName = "Assistant"
		}
		fmt.Fprintf(&sb, "%s: %s\n\n", roleName, m.Content)
	}

	sb.WriteString("Please formulate a concrete plan and execute the requested work.")
	return sb.String()
}
