package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

// ForApp returns the project a connected app works in, making it the first time. Reconnecting the
// same app, or connecting it twice at once, always lands in the same project.
func (s *Service) ForApp(ctx context.Context, userID, app string) (domain.Project, error) {
	source, _, err := domain.Normalize(app, "")
	if err != nil {
		return domain.Project{}, fmt.Errorf("name the app project: %w", err)
	}

	project, err := s.store.FindBySource(ctx, userID, source)
	if err == nil || !errors.Is(err, domain.ErrNotFound) {
		return project, wrapFind(err)
	}

	project, err = s.create(ctx, userID, source, "Made for "+source, source)
	if errors.Is(err, domain.ErrSourceTaken) {
		return s.ForApp(ctx, userID, app)
	}

	return project, err
}

func wrapFind(err error) error {
	if err != nil {
		return fmt.Errorf("find app project: %w", err)
	}

	return nil
}
