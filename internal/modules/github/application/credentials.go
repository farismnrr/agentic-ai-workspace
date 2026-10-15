package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

// Credentials returns what a command run for this person needs, refreshing the token when it is
// close to ending. A person who never connected, or whose grant GitHub no longer accepts, gets
// ErrNotConnected. It only ever returns the credentials of the person it is asked about.
func (s *Service) Credentials(ctx context.Context, userID string) (domain.Credential, error) {
	if !s.enabled {
		return domain.Credential{}, domain.ErrNotConnected
	}

	unlock := s.lock(userID)
	defer unlock()

	return s.current(ctx, userID)
}

// current must be called with the person's lock held.
func (s *Service) current(ctx context.Context, userID string) (domain.Credential, error) {
	rec, ok, err := s.store.Load(ctx, userID)
	if err != nil {
		return domain.Credential{}, fmt.Errorf("load connection: %w", err)
	}

	if !ok {
		return domain.Credential{}, domain.ErrNotConnected
	}

	tokens, err := s.open(rec)
	if err != nil {
		return domain.Credential{}, err
	}

	if tokens.ExpiresWithin(s.now(), refreshMargin) {
		if tokens, err = s.renew(ctx, rec, tokens); err != nil {
			return domain.Credential{}, err
		}
	}

	return domain.Credential{Token: tokens.Access, Profile: rec.Profile}, nil
}

// renew trades the refresh token for new tokens. When GitHub rejects it the connection is gone.
// A failure to reach GitHub keeps the old token while it still works.
func (s *Service) renew(ctx context.Context, rec Record, old domain.Tokens) (domain.Tokens, error) {
	fresh, err := s.provider.Refresh(ctx, old.Refresh)
	if errors.Is(err, domain.ErrRejected) {
		_ = s.store.Delete(ctx, rec.UserID)

		return domain.Tokens{}, domain.ErrNotConnected
	}

	if err != nil {
		if old.AccessExpires.After(s.now()) {
			return old, nil
		}

		return domain.Tokens{}, fmt.Errorf("refresh token: %w", err)
	}

	if err = s.save(ctx, rec.UserID, rec.Profile, fresh); err != nil {
		return domain.Tokens{}, err
	}

	return fresh, nil
}

func (s *Service) open(rec Record) (domain.Tokens, error) {
	access, err := s.sealer.Decrypt(rec.AccessSealed)
	if err != nil {
		return domain.Tokens{}, fmt.Errorf("open access token: %w", err)
	}

	refresh, err := s.sealer.Decrypt(rec.RefreshSealed)
	if err != nil {
		return domain.Tokens{}, fmt.Errorf("open refresh token: %w", err)
	}

	return domain.Tokens{
		Access: string(access), Refresh: string(refresh),
		AccessExpires: rec.AccessExpires, RefreshExpires: rec.RefreshExpires,
	}, nil
}

func (s *Service) save(ctx context.Context, userID string, profile domain.Profile, t domain.Tokens) error {
	access, err := s.sealer.Encrypt([]byte(t.Access))
	if err != nil {
		return fmt.Errorf("seal access token: %w", err)
	}

	refresh, err := s.sealer.Encrypt([]byte(t.Refresh))
	if err != nil {
		return fmt.Errorf("seal refresh token: %w", err)
	}

	rec := Record{
		UserID: userID, Profile: profile, AccessSealed: access, RefreshSealed: refresh,
		AccessExpires: t.AccessExpires, RefreshExpires: t.RefreshExpires,
	}
	if err = s.store.Save(ctx, rec, s.now()); err != nil {
		return fmt.Errorf("save connection: %w", err)
	}

	return nil
}
