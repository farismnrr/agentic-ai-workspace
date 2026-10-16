package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/domain"
)

// Store persists a person's team.
type Store interface {
	// Load returns the saved team; found is false when the person never saved one.
	Load(ctx context.Context, userID string) (team domain.Team, found bool, err error)
	Save(ctx context.Context, userID string, team domain.Team, at time.Time) error
	Clear(ctx context.Context, userID string) error
}

// Service holds the agent team use cases.
type Service struct {
	store Store
	now   func() time.Time
}

// NewService wires the use cases.
func NewService(store Store, now func() time.Time) *Service {
	return &Service{store: store, now: now}
}

// Get returns a person's team: their saved one, or the built-in SDLC team.
func (s *Service) Get(ctx context.Context, userID string) (domain.Team, error) {
	team, found, err := s.store.Load(ctx, userID)
	if err != nil {
		return domain.Team{}, fmt.Errorf("load agent team: %w", err)
	}

	if !found {
		return domain.Presets(), nil
	}

	// A team saved before a built-in role existed still gets it.
	return team.WithBuiltIns(), nil
}

// Update replaces a person's team.
func (s *Service) Update(ctx context.Context, userID string, team domain.Team) (domain.Team, error) {
	clean, err := team.Normalize()
	if err != nil {
		return domain.Team{}, fmt.Errorf("update agent team: %w", err)
	}

	if err := s.store.Save(ctx, userID, clean, s.now()); err != nil {
		return domain.Team{}, fmt.Errorf("save agent team: %w", err)
	}

	return clean, nil
}

// Reset forgets a person's team, so they have the built-in one again.
func (s *Service) Reset(ctx context.Context, userID string) (domain.Team, error) {
	if err := s.store.Clear(ctx, userID); err != nil {
		return domain.Team{}, fmt.Errorf("reset agent team: %w", err)
	}

	return domain.Presets(), nil
}
