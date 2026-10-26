package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

// Cipher encrypts API keys at rest.
type Cipher interface {
	Encrypt(plain []byte) (string, error)
	Decrypt(sealed string) ([]byte, error)
}

// Store persists providers and their sealed keys.
type Store interface {
	Create(ctx context.Context, p domain.Provider, sealedKey string) error
	List(ctx context.Context, userID string) ([]domain.Provider, error)
	Find(ctx context.Context, userID, id string) (domain.Provider, string, error)
	SetVerified(ctx context.Context, userID, id string, verified bool, models []domain.ModelInfo) error
	SetEnabled(ctx context.Context, userID, id string, enabled bool) error
	SetContextWindow(ctx context.Context, userID, id, model string, tokens int64) error
	Delete(ctx context.Context, userID, id string) error
}

// Verifier asks a provider which models a key can use; success proves the key works.
type Verifier interface {
	Models(ctx context.Context, credential domain.Credential) ([]domain.ModelInfo, error)
}

// IDs mints identifiers.
type IDs interface {
	ID() (string, error)
}

// Service holds the provider use cases.
type Service struct {
	now      func() time.Time
	store    Store
	cipher   Cipher
	verifier Verifier
	ids      IDs
	plans    Plans
	// synced remembers when each gateway's model list was last read, so it is not read on every page load.
	synced map[string]time.Time
	mu     sync.Mutex
}

// NewService wires the use cases.
func NewService(store Store, cipher Cipher, verifier Verifier, ids IDs, now func() time.Time) *Service {
	return &Service{store: store, cipher: cipher, verifier: verifier, ids: ids, now: now, synced: map[string]time.Time{}}
}

// Verify checks the key with the provider and caches the models it offers.
func (s *Service) Verify(ctx context.Context, userID, id string) (domain.Provider, error) {
	if domain.IsPlanID(id) {
		return s.plan(ctx, userID, id)
	}

	credential, err := s.Credential(ctx, userID, id)
	if err != nil {
		return domain.Provider{}, err
	}

	models, verifyErr := s.verifier.Models(ctx, credential)
	if verifyErr != nil {
		models = nil
	}

	if err = s.store.SetVerified(ctx, userID, id, verifyErr == nil, models); err != nil {
		return domain.Provider{}, fmt.Errorf("save verification: %w", err)
	}

	if verifyErr != nil {
		return domain.Provider{}, errors.Join(domain.ErrVerifyFailed, verifyErr)
	}

	provider, _, err := s.store.Find(ctx, userID, id)
	if err != nil {
		return domain.Provider{}, fmt.Errorf("reload provider: %w", err)
	}

	return provider, nil
}

// Credential decrypts what a model client needs; it is only for server-side use.
func (s *Service) Credential(ctx context.Context, userID, id string) (domain.Credential, error) {
	if domain.IsPlanID(id) {
		return domain.Credential{}, fmt.Errorf("a plan has no key: %w", domain.ErrNotFound)
	}

	provider, sealed, err := s.store.Find(ctx, userID, id)
	if err != nil {
		return domain.Credential{}, fmt.Errorf("find provider: %w", err)
	}

	if domain.IsLegacy9router(provider.Kind, provider.Name, provider.BaseURL) {
		return domain.Credential{}, domain.ErrAccountRelay
	}

	key, err := s.cipher.Decrypt(sealed)
	if err != nil {
		return domain.Credential{}, fmt.Errorf("open key: %w", err)
	}

	return domain.Credential{Kind: provider.Kind, BaseURL: provider.BaseURL, APIKey: string(key)}, nil
}

// List returns a person's providers.
func (s *Service) List(ctx context.Context, userID string) ([]domain.Provider, error) {
	providers, err := s.store.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	disableLegacyAccountRelays(providers)
	if s.plans != nil {
		providers = append(s.plans.Connected(ctx, userID), providers...)
	}

	return providers, nil
}

// SetEnabled switches a provider on or off.
func (s *Service) SetEnabled(ctx context.Context, userID, id string, enabled bool) error {
	if domain.IsPlanID(id) && s.plans != nil {
		if err := s.plans.SetEnabled(ctx, userID, id, enabled); err != nil {
			return fmt.Errorf("switch plan: %w", err)
		}

		return nil
	}

	if enabled {
		if err := s.checkLegacyAccountRelay(ctx, userID, id); err != nil {
			return err
		}
	}
	if err := s.store.SetEnabled(ctx, userID, id, enabled); err != nil {
		return fmt.Errorf("switch provider: %w", err)
	}

	return nil
}

// Remove deletes a provider and its key.
func (s *Service) Remove(ctx context.Context, userID, id string) error {
	if domain.IsPlanID(id) && s.plans != nil {
		if err := s.plans.SignOut(ctx, userID, id); err != nil {
			return fmt.Errorf("sign out of plan: %w", err)
		}

		return nil
	}

	if err := s.store.Delete(ctx, userID, id); err != nil {
		return fmt.Errorf("remove provider: %w", err)
	}

	return nil
}

// ModelOption is one model a person can pick.
type ModelOption struct {
	ProviderID   string
	ProviderName string
	Model        string
	// Group is who serves the model when the gateway reports a provider alias.
	Group string
}

// Models lists the models of every enabled, verified provider.
func (s *Service) Models(ctx context.Context, userID string) ([]ModelOption, error) {
	providers, err := s.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	options := []ModelOption{}

	for _, provider := range providers {
		if !provider.Enabled || !provider.Verified {
			continue
		}

		for _, model := range provider.Models {
			options = append(options, ModelOption{
				ProviderID: provider.ID, ProviderName: provider.Name, Model: model, Group: provider.GroupOf(model),
			})
		}
	}

	return options, nil
}
