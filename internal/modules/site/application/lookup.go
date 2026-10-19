package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func isNotFound(err error) bool { return errors.Is(err, domain.ErrNotFound) }

// List returns a person's sites. A site whose container is gone is marked stopped on the way.
func (s *Service) List(ctx context.Context, userID string) ([]domain.Site, error) {
	sites, err := s.deps.Store.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}

	for i, site := range sites {
		if site.Status == domain.Running && !s.deps.Runtime.Alive(ctx, site.ContainerID) {
			_ = s.deps.Store.MarkStopped(ctx, site.ProjectID)
			sites[i].Status = domain.Stopped
		}
	}

	return sites, nil
}

// Resolve finds the running site an address name belongs to. Unknown and stopped names
// both answer ErrNotFound, so nothing reveals which exist.
func (s *Service) Resolve(ctx context.Context, slug string) (domain.Site, error) {
	return s.running(s.deps.Store.BySlug(ctx, slug))
}

// ResolveOld finds the running site an address from before names were unique points to: the
// token in its second segment decides, whatever the name says.
func (s *Service) ResolveOld(ctx context.Context, token string) (domain.Site, error) {
	return s.running(s.deps.Store.ByToken(ctx, token))
}

func (s *Service) running(site domain.Site, err error) (domain.Site, error) {
	if err != nil || site.Status != domain.Running {
		return domain.Site{}, domain.ErrNotFound
	}

	return site, nil
}

// Socket is where the site's app can be reached.
func (s *Service) Socket(projectID string) string { return s.deps.Runtime.Socket(projectID) }

// Reconcile runs at startup: containers do not survive a restart, so nothing is running.
func (s *Service) Reconcile(ctx context.Context) error {
	if err := s.deps.Store.MarkStopped(ctx, ""); err != nil {
		return fmt.Errorf("reset sites: %w", err)
	}

	return nil
}
