package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// ChatOnExternal routes a person's message to the internal conversation attached to an external task.
func (s *Service) ChatOnExternal(ctx context.Context, userID, externalID, message string) (string, error) {
	message = strings.TrimSpace(message)
	if err := validatePrompt(message); err != nil {
		return "", err
	}
	external, err := s.owned(ctx, userID, externalID)
	if err != nil {
		return "", err
	}
	if external.ArchivedAt != nil || (external.Source == "" && external.WorkID == "") {
		return "", domain.ErrWrongState
	}
	if linkedID, handled, err := s.followExistingLinkedChat(ctx, userID, external, message); handled || err != nil {
		return linkedID, err
	}

	return s.createLinkedChat(ctx, external, message)
}

func (s *Service) followExistingLinkedChat(
	ctx context.Context,
	userID string,
	external domain.Task,
	message string,
) (string, bool, error) {
	if external.LinkedTaskID == "" {
		return "", false, nil
	}
	linked, err := s.owned(ctx, userID, external.LinkedTaskID)
	if err != nil {
		return "", true, err
	}
	invalidLink := linked.ProjectID != external.ProjectID || linked.OriginTaskID != external.ID ||
		linked.Source != "" || linked.WorkID != ""
	if invalidLink {
		return "", true, domain.ErrNotFound
	}
	if linked.ArchivedAt != nil {
		return "", false, nil
	}
	if err := s.followLinkedChat(ctx, external, message); err != nil {
		return "", true, err
	}
	return linked.ID, true, nil
}

func (s *Service) createLinkedChat(ctx context.Context, external domain.Task, message string) (string, error) {
	prompt, err := s.externalContext(ctx, external, message)
	if err != nil {
		return "", err
	}
	task, event, err := s.buildLinkedTask(ctx, external, message)
	if err != nil {
		return "", err
	}
	linked, created, err := s.deps.LinkedTasks.CreateLinkedTask(ctx, external.ID, task, event, prompt)
	if err != nil {
		return "", fmt.Errorf("create linked chat: %w", err)
	}
	if created {
		if s.deps.Notify != nil {
			s.deps.Notify.Changed(linked.UserID, linked.ID)
			s.deps.Notify.Changed(external.UserID, external.ID)
		}
		s.dispatcher.Enqueue(linked.ID, linked.ProjectID)
	} else {
		external.LinkedTaskID = linked.ID
		if err := s.followLinkedChat(ctx, external, message); err != nil {
			return "", err
		}
	}
	return linked.ID, nil
}

func (s *Service) followLinkedChat(ctx context.Context, external domain.Task, message string) error {
	linked, err := s.owned(ctx, external.UserID, external.LinkedTaskID)
	if err != nil {
		return err
	}
	invalidLink := linked.ProjectID != external.ProjectID || linked.OriginTaskID != external.ID ||
		linked.Source != "" || linked.WorkID != ""
	if invalidLink {
		return domain.ErrNotFound
	}
	return s.followUp(ctx, external.UserID, linked.ID, FollowUpInput{Text: message}, false)
}
