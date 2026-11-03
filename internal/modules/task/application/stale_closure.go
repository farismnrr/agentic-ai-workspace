package application

import (
	"context"
	"fmt"
)

func (s *Service) closeStaleNeedsInput(ctx context.Context, userID, taskID string) error {
	if err := s.deps.Tasks.CloseStaleNeedsInput(ctx, userID, taskID, s.now()); err != nil {
		return fmt.Errorf("close stale task: %w", err)
	}
	return nil
}
