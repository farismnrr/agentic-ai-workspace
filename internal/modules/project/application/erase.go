package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

// Delete removes a project for good: what depends on it first, then the project, then its folder. It cannot be undone.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	if _, err := s.store.Find(ctx, userID, id); err != nil {
		return fmt.Errorf("find project: %w", err)
	}

	for _, clean := range s.beforeDelete {
		if err := clean(ctx, userID, id); err != nil {
			return fmt.Errorf("clean up before deleting the project: %w", err)
		}
	}

	if err := s.store.Delete(ctx, userID, id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	if err := s.folders.Remove(id); err != nil {
		return fmt.Errorf("remove project folder: %w", err)
	}

	return nil
}

// BeforeDelete registers what must be removed before a project can be deleted (its tasks, its running site).
func (s *Service) BeforeDelete(f func(ctx context.Context, userID, id string) error) {
	s.beforeDelete = append(s.beforeDelete, f)
}

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
