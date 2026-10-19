package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

// Available says whether a name is free for a new project (or for the project exceptID, when renaming) and,
// when it is not, the nearest name that is.
func (s *Service) Available(ctx context.Context, name, exceptID string) (free bool, suggestion string, err error) {
	name, _, err = domain.Normalize(name, "")
	if err != nil {
		return false, "", fmt.Errorf("check name: %w", err)
	}

	existing, err := s.store.ActiveNames(ctx)
	if err != nil {
		return false, "", fmt.Errorf("check name: %w", err)
	}

	return !domain.Taken(name, exceptID, existing), domain.Suggest(name, exceptID, existing), nil
}

func (s *Service) requireFree(ctx context.Context, name, exceptID string) error {
	free, _, err := s.Available(ctx, name, exceptID)
	if err != nil {
		return err
	}

	if !free {
		return domain.ErrNameTaken
	}

	return nil
}
