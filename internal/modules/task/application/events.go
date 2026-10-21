package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Events returns the thread entries after a cursor.
func (s *Service) Events(ctx context.Context, userID, taskID string, after int64) ([]domain.Event, error) {
	if _, err := s.owned(ctx, userID, taskID); err != nil {
		return nil, err
	}
	events, err := s.deps.Events.EventsAfter(ctx, taskID, after)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return events, nil
}
