package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
)

// SessionTTL is how long a browser session stays valid.
const SessionTTL = 30 * 24 * time.Hour

// Session is a freshly issued browser session.
type Session struct {
	ExpiresAt time.Time
	Token     string
}

// Service holds the account use cases.
type Service struct {
	users      UserStore
	sessions   SessionStore
	hasher     PasswordHasher
	tokens     Tokens
	now        func() time.Time
	adminEmail string
	erasers    []Eraser
}

// NewService wires the account use cases.
func NewService(
	users UserStore, sessions SessionStore, hasher PasswordHasher, tokens Tokens, now func() time.Time, adminEmail string,
) *Service {
	return &Service{users: users, sessions: sessions, hasher: hasher, tokens: tokens, now: now, adminEmail: adminEmail}
}

// Register creates an account and signs it in.
func (s *Service) Register(ctx context.Context, name, email, password string) (domain.User, Session, error) {
	cleanName, err := domain.NormalizeName(name)
	if err != nil {
		return domain.User{}, Session{}, fmt.Errorf("register: %w", err)
	}

	cleanEmail, err := domain.NormalizeEmail(email)
	if err != nil {
		return domain.User{}, Session{}, fmt.Errorf("register: %w", err)
	}

	if err = domain.ValidatePassword(password); err != nil {
		return domain.User{}, Session{}, fmt.Errorf("register: %w", err)
	}

	user, err := s.create(ctx, cleanName, cleanEmail, password)
	if err != nil {
		return domain.User{}, Session{}, err
	}

	session, err := s.openSession(ctx, user.ID)

	return user, session, err
}

func (s *Service) create(ctx context.Context, name, email, password string) (domain.User, error) {
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	id, err := s.tokens.ID()
	if err != nil {
		return domain.User{}, fmt.Errorf("new user id: %w", err)
	}

	role := domain.RoleUser
	if email == s.adminEmail {
		role = domain.RoleAdmin
	}
	user := domain.User{ID: id, Name: name, Email: email, Role: role, CreatedAt: s.now()}
	if err := s.users.Create(ctx, user, hash); err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
