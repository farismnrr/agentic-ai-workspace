package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

const (
	stateTTL      = 10 * time.Minute
	refreshMargin = 10 * time.Minute
)

type pending struct {
	expires  time.Time
	userID   string
	verifier string
}

// Service connects people's GitHub accounts and hands their credentials to the commands they run.
type Service struct {
	store    Store
	provider Provider
	sealer   Sealer
	secrets  Secrets
	logs     LogSource
	now      func() time.Time
	states   map[string]pending
	perUser  map[string]*sync.Mutex
	created  map[string]map[string]time.Time
	mu       sync.Mutex
	enabled  bool
}

// NewService wires the use cases; enabled is false when the GitHub App is not configured.
func NewService(
	store Store, provider Provider, sealer Sealer, secrets Secrets, now func() time.Time, enabled bool,
) *Service {
	return &Service{
		store: store, provider: provider, sealer: sealer, secrets: secrets, now: now, enabled: enabled,
		states: map[string]pending{}, perUser: map[string]*sync.Mutex{}, created: map[string]map[string]time.Time{},
	}
}

// Enabled reports whether people can connect GitHub at all.
func (s *Service) Enabled() bool {
	return s.enabled
}

// Start begins a sign-in for one person and returns the GitHub address to send them to. The state
// is single use and tied to that person, so an answer can never be completed by anyone else.
func (s *Service) Start(userID string) (string, error) {
	if !s.enabled {
		return "", domain.ErrNotConfigured
	}

	state, _, err := s.secrets.New()
	if err != nil {
		return "", fmt.Errorf("mint state: %w", err)
	}

	verifier, _, err := s.secrets.New()
	if err != nil {
		return "", fmt.Errorf("mint verifier: %w", err)
	}

	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, p := range s.states {
		if p.expires.Before(now) {
			delete(s.states, key)
		}
	}

	s.states[state] = pending{userID: userID, verifier: verifier, expires: now.Add(stateTTL)}

	return s.provider.AuthorizeURL(state, challenge(verifier)), nil
}

// take consumes a state; it only works once, before it expires, for the person who started it.
func (s *Service) take(userID, state string) (pending, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.states[state]
	delete(s.states, state)

	if !ok || p.userID != userID || p.expires.Before(s.now()) {
		return pending{}, false
	}

	return p, true
}

// Finish completes the sign-in GitHub sent the person back from.
func (s *Service) Finish(ctx context.Context, userID, code, state string) error {
	p, ok := s.take(userID, state)
	if !ok {
		return domain.ErrState
	}

	tokens, err := s.provider.Exchange(ctx, code, p.verifier)
	if err != nil {
		return fmt.Errorf("exchange code: %w", err)
	}

	profile, err := s.provider.Profile(ctx, tokens.Access)
	if err != nil {
		return fmt.Errorf("read profile: %w", err)
	}

	unlock := s.lock(userID)
	defer unlock()

	return s.save(ctx, userID, profile, tokens)
}

// Status returns the person's connection, or ErrNotConnected.
func (s *Service) Status(ctx context.Context, userID string) (domain.Connection, error) {
	rec, ok, err := s.store.Load(ctx, userID)
	if err != nil {
		return domain.Connection{}, fmt.Errorf("load connection: %w", err)
	}

	if !ok {
		return domain.Connection{}, domain.ErrNotConnected
	}

	return domain.Connection{Profile: rec.Profile}, nil
}

// Disconnect forgets the connection and asks GitHub to revoke the grant.
func (s *Service) Disconnect(ctx context.Context, userID string) error {
	unlock := s.lock(userID)
	defer unlock()

	if cred, err := s.current(ctx, userID); err == nil {
		_ = s.provider.Revoke(ctx, cred.Token)
	}

	if err := s.store.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}

	return nil
}

// lock serialises work on one person's tokens: GitHub replaces the refresh token each time it is
// used, so two refreshes at once would leave one of them holding a dead token.
func (s *Service) lock(userID string) func() {
	s.mu.Lock()

	m, ok := s.perUser[userID]
	if !ok {
		m = &sync.Mutex{}
		s.perUser[userID] = m
	}

	s.mu.Unlock()
	m.Lock()

	return m.Unlock
}
