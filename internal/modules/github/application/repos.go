package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

// Repos lists the repositories the person let the app reach. It needs the connection, and it returns the
// repositories of the person it is asked about only.
func (s *Service) Repos(ctx context.Context, userID string) ([]domain.Repo, error) {
	cred, err := s.Credentials(ctx, userID)
	if err != nil {
		return nil, err
	}

	repos, err := s.provider.Repos(ctx, cred.Token)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}

	return repos, nil
}
