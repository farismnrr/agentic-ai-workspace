package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Shot returns a screenshot of one of the person's tasks.
func (s *Service) Shot(ctx context.Context, userID, taskID, name string) ([]byte, error) {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}

	if s.deps.Shots == nil {
		return nil, domain.ErrNotFound
	}

	data, err := s.deps.Shots.Load(ctx, task.ID, name)
	if err != nil {
		return nil, fmt.Errorf("load screenshot: %w", err)
	}

	return data, nil
}
