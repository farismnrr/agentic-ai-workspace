package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/domain"
)

// Store persists a person's preferences.
type Store interface {
	Load(ctx context.Context, userID string) (domain.Preferences, error)
	Save(ctx context.Context, userID string, prefs domain.Preferences, at time.Time) error
}

// IDs mints identifiers for new rules.
type IDs interface {
	ID() (string, error)
}

// Service holds the preference use cases.
type Service struct {
	store Store
	ids   IDs
	now   func() time.Time
}

// NewService wires the use cases.
func NewService(store Store, ids IDs, now func() time.Time) *Service {
	return &Service{store: store, ids: ids, now: now}
}

// Get returns a person's preferences, empty when they never saved any.
func (s *Service) Get(ctx context.Context, userID string) (domain.Preferences, error) {
	prefs, err := s.store.Load(ctx, userID)
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("load preferences: %w", err)
	}

	return prefs, nil
}

// Update replaces a person's preferences; new rules get identifiers.
func (s *Service) Update(ctx context.Context, userID string, prefs domain.Preferences) (domain.Preferences, error) {
	clean, err := prefs.Normalize()
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("update preferences: %w", err)
	}

	for i := range clean.Rules {
		if clean.Rules[i].ID != "" {
			continue
		}

		if clean.Rules[i].ID, err = s.ids.ID(); err != nil {
			return domain.Preferences{}, fmt.Errorf("new rule id: %w", err)
		}
	}

	if err := s.store.Save(ctx, userID, clean, s.now()); err != nil {
		return domain.Preferences{}, fmt.Errorf("save preferences: %w", err)
	}

	return clean, nil
}
