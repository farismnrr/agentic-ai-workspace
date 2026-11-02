package application

import (
	"context"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (s *Service) SaveExternalResearch(ctx context.Context, userID, taskID, markdown string) error {
	task, err := s.externalOwned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(markdown) == "" || len(markdown) > 12000 {
		return domain.ErrBadRequest
	}
	updated := appendResearch(task.Research, markdown)
	if err := s.deps.Tasks.SaveResearch(ctx, task.ID, updated, s.now()); err != nil {
		return err
	}
	_, err = s.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventResearch, Text: "External lead saved research",
		Technical: markdown, Agent: task.Source, Time: s.now(),
	})
	return err
}

func (s *Service) SaveExternalPRD(ctx context.Context, userID, taskID, markdown string) error {
	task, err := s.externalOwned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(markdown) == "" || len(markdown) > 12000 {
		return domain.ErrBadRequest
	}
	if err := s.deps.Tasks.SavePRD(ctx, task.ID, markdown, s.now()); err != nil {
		return err
	}
	_, err = s.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventPRD, Text: "External lead saved the PRD",
		Technical: markdown, Agent: task.Source, Time: s.now(),
	})
	return err
}
