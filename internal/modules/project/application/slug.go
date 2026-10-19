package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

const maxSlugTries = 1000

// SlugStatus is the answer to "can this be the project ID?".
type SlugStatus struct {
	// Suggestion is the ID to offer: the one asked about when it is free, else the nearest free one.
	Suggestion string
	Valid      bool
	Available  bool
}

// CheckSlug says whether slug can be a new project's ID. When slug is empty the ID is made from name, the way
// the form fills it in while the name is typed.
func (s *Service) CheckSlug(ctx context.Context, slug, name string) (SlugStatus, error) {
	if slug == "" {
		slug = domain.SlugFromName(name)
	}

	if !domain.ValidSlug(slug) {
		return SlugStatus{Suggestion: domain.SlugFromName(slug)}, nil
	}

	taken, err := s.store.SlugTaken(ctx, slug)
	if err != nil {
		return SlugStatus{}, fmt.Errorf("check project id: %w", err)
	}

	if !taken {
		return SlugStatus{Suggestion: slug, Valid: true, Available: true}, nil
	}

	free, err := s.freeSlug(ctx, slug)

	return SlugStatus{Suggestion: free, Valid: true}, err
}

// chooseSlug returns the ID a new project gets: the one the person gave (refused when invalid or taken), or a free
// one made from the name.
func (s *Service) chooseSlug(ctx context.Context, slug, name string) (string, error) {
	if slug == "" {
		return s.freeSlug(ctx, domain.SlugFromName(name))
	}

	if !domain.ValidSlug(slug) {
		return "", domain.ErrInvalidSlug
	}

	taken, err := s.store.SlugTaken(ctx, slug)
	if err != nil {
		return "", fmt.Errorf("check project id: %w", err)
	}

	if taken {
		return "", domain.ErrSlugTaken
	}

	return slug, nil
}

// freeSlug returns base, or base with the first number that makes it free.
func (s *Service) freeSlug(ctx context.Context, base string) (string, error) {
	for n := 1; n <= maxSlugTries; n++ {
		slug := base
		if n > 1 {
			slug = domain.WithNumber(base, n)
		}

		taken, err := s.store.SlugTaken(ctx, slug)
		if err != nil {
			return "", fmt.Errorf("check project id: %w", err)
		}

		if !taken {
			return slug, nil
		}
	}

	return "", domain.ErrSlugTaken
}
