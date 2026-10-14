package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
)

// dummyHash lets Login spend comparable time for unknown emails.
const dummyHash = "$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

// Login verifies credentials and opens a session.
func (s *Service) Login(ctx context.Context, email, password string) (domain.User, Session, error) {
	cleanEmail, err := domain.NormalizeEmail(email)
	if err != nil {
		return domain.User{}, Session{}, domain.ErrInvalidCredentials
	}

	user, hash, err := s.users.ByEmail(ctx, cleanEmail)
	if errors.Is(err, domain.ErrUserNotFound) {
		s.hasher.Verify(password, dummyHash)
		return domain.User{}, Session{}, domain.ErrInvalidCredentials
	}

	if err != nil {
		return domain.User{}, Session{}, fmt.Errorf("find user: %w", err)
	}

	if !s.hasher.Verify(password, hash) {
		return domain.User{}, Session{}, domain.ErrInvalidCredentials
	}

	session, err := s.openSession(ctx, user.ID)

	return user, session, err
}

// Authenticate resolves a session cookie token to its user.
func (s *Service) Authenticate(ctx context.Context, token string) (domain.User, error) {
	userID, err := s.sessions.SessionUserID(ctx, s.tokens.Hash(token), s.now())
	if err != nil {
		return domain.User{}, fmt.Errorf("find session: %w", err)
	}

	user, err := s.users.ByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("find session user: %w", err)
	}

	return user, nil
}

// Logout ends a session. Unknown tokens are ignored.
func (s *Service) Logout(ctx context.Context, token string) error {
	if err := s.sessions.DeleteSession(ctx, s.tokens.Hash(token)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (s *Service) openSession(ctx context.Context, userID string) (Session, error) {
	plain, hash, err := s.tokens.New()
	if err != nil {
		return Session{}, fmt.Errorf("new session token: %w", err)
	}

	expires := s.now().Add(SessionTTL)
	if err := s.sessions.CreateSession(ctx, hash, userID, expires); err != nil {
		return Session{}, fmt.Errorf("store session: %w", err)
	}

	return Session{Token: plain, ExpiresAt: expires}, nil
}

// Profile returns a user by id, for features that only hold the id.
func (s *Service) Profile(ctx context.Context, id string) (domain.User, error) {
	user, err := s.users.ByID(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("find user: %w", err)
	}

	return user, nil
}
