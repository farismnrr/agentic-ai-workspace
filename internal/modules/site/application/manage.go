package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// Stop stops a site its owner asks about.
func (s *Service) Stop(ctx context.Context, userID, projectID string) error {
	site, err := s.owned(ctx, userID, projectID)
	if err != nil {
		return err
	}

	return s.stop(ctx, site)
}

// Restart starts a site again the way it was last started, for the person who asks. The address stays the same.
func (s *Service) Restart(ctx context.Context, userID, projectID string) (domain.Site, error) {
	site, err := s.owned(ctx, userID, projectID)
	if err != nil {
		return domain.Site{}, err
	}

	if site.Command == "" {
		return domain.Site{}, domain.ErrNeverStarted
	}

	return s.Start(ctx, projectID, site.Command, site.Port)
}

// StopProject stops a project's site when the project is archived or deleted.
func (s *Service) StopProject(ctx context.Context, projectID string) {
	if site, err := s.deps.Store.ByProject(ctx, projectID); err == nil {
		_ = s.stop(ctx, site)
	}
}

// StopUser stops every site of a person, for when their account is deleted.
func (s *Service) StopUser(ctx context.Context, userID string) error {
	sites, err := s.deps.Store.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list sites: %w", err)
	}

	for _, site := range sites {
		if err = s.stop(ctx, site); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) stop(ctx context.Context, site domain.Site) error {
	if site.Status == domain.Running {
		s.deps.Runtime.Stop(ctx, site.ContainerID)
	}

	if err := s.deps.Store.MarkStopped(ctx, site.ProjectID); err != nil {
		return fmt.Errorf("mark site stopped: %w", err)
	}

	s.deps.Notify.Changed(site.UserID)

	return nil
}

// SetVisibility changes who may open a site. Only its owner can.
func (s *Service) SetVisibility(ctx context.Context, userID, projectID, value string) error {
	visibility, err := domain.ParseVisibility(value)
	if err != nil {
		return fmt.Errorf("set visibility: %w", err)
	}

	site, err := s.owned(ctx, userID, projectID)
	if err != nil {
		return err
	}

	if err = s.deps.Store.SetVisibility(ctx, site.ProjectID, visibility); err != nil {
		return fmt.Errorf("set visibility: %w", err)
	}

	s.deps.Notify.Changed(userID)

	return nil
}

func (s *Service) owned(ctx context.Context, userID, projectID string) (domain.Site, error) {
	site, err := s.deps.Store.ByProject(ctx, projectID)
	if err != nil || site.UserID != userID {
		return domain.Site{}, domain.ErrNotFound
	}

	return site, nil
}
