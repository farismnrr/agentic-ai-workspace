package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

// DeleteAll removes all of a person's projects and their folders. Tasks must be gone first.
func (s *Service) DeleteAll(ctx context.Context, userID string) error {
	ids, err := s.store.DeleteAll(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete projects: %w", err)
	}

	for _, id := range ids {
		if err := s.folders.Remove(id); err != nil {
			return fmt.Errorf("remove project folder: %w", err)
		}
	}

	return nil
}

// Everything lists all of a person's projects, archived ones too.
func (s *Service) Everything(ctx context.Context, userID string) ([]domain.Project, error) {
	projects, err := s.store.ListAll(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	return projects, nil
}
