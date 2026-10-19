package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// address returns the token and the address name of a project's site: the ones it already has, or new ones.
func (s *Service) address(
	ctx context.Context, previous domain.Site, projectID, name string,
) (token, slug string, err error) {
	token, slug = previous.Token, previous.Slug

	if token == "" {
		if token, err = s.deps.IDs.UUID(); err != nil {
			return "", "", fmt.Errorf("make site address: %w", err)
		}
	}

	if slug == "" {
		if slug, err = s.freeSlug(ctx, projectID, name); err != nil {
			return "", "", err
		}
	}

	return token, slug, nil
}

// freeSlug picks the address name for a project's first start: its name, or its name with the
// first number that is free. Other projects' names are never taken.
func (s *Service) freeSlug(ctx context.Context, projectID, name string) (string, error) {
	base := domain.Slugify(name)

	for n := 1; n <= maxSlugTries; n++ {
		slug := base
		if n > 1 {
			slug = fmt.Sprintf("%s-%d", base, n)
		}

		other, err := s.deps.Store.BySlug(ctx, slug)

		switch {
		case isNotFound(err):
			return slug, nil
		case err != nil:
			return "", fmt.Errorf("check site address: %w", err)
		case other.ProjectID == projectID:
			return slug, nil
		}
	}

	return "", domain.ErrNameTaken
}
