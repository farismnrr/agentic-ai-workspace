package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

const maxProviders = 10

// AddInput is what a person enters in the Add provider form.
type AddInput struct {
	Kind    domain.Kind
	Name    string
	BaseURL string
	APIKey  string
}

// Add stores a provider with its key sealed; it still needs verifying.
func (s *Service) Add(ctx context.Context, userID string, in AddInput) (domain.Provider, error) {
	provider, err := s.build(userID, in)
	if err != nil {
		return domain.Provider{}, err
	}

	existing, err := s.store.List(ctx, userID)
	if err != nil {
		return domain.Provider{}, fmt.Errorf("list providers: %w", err)
	}

	if len(existing) >= maxProviders {
		return domain.Provider{}, domain.ErrTooMany
	}

	sealed, err := s.cipher.Encrypt([]byte(strings.TrimSpace(in.APIKey)))
	if err != nil {
		return domain.Provider{}, fmt.Errorf("seal key: %w", err)
	}

	if err := s.store.Create(ctx, provider, sealed); err != nil {
		return domain.Provider{}, fmt.Errorf("save provider: %w", err)
	}

	return provider, nil
}

func (s *Service) build(userID string, in AddInput) (domain.Provider, error) {
	if !in.Kind.Valid() {
		return domain.Provider{}, domain.ErrInvalidKind
	}

	name, err := domain.NormalizeName(in.Name)
	if err != nil {
		return domain.Provider{}, fmt.Errorf("add provider: %w", err)
	}

	baseURL, err := domain.NormalizeBaseURL(in.Kind, in.BaseURL)
	if err != nil {
		return domain.Provider{}, fmt.Errorf("add provider: %w", err)
	}

	key := strings.TrimSpace(in.APIKey)
	if key == "" {
		return domain.Provider{}, domain.ErrInvalidKey
	}

	id, err := s.ids.ID()
	if err != nil {
		return domain.Provider{}, fmt.Errorf("new provider id: %w", err)
	}

	return domain.Provider{
		ID: id, UserID: userID, Kind: in.Kind, Name: name, BaseURL: baseURL, KeyHint: domain.KeyHint(key),
		Enabled: true, CreatedAt: s.now(), Models: []string{},
	}, nil
}
