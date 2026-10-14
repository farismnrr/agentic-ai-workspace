package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
)

// UserStore persists accounts.
type UserStore interface {
	Create(ctx context.Context, user domain.User, passwordHash string) error
	ByEmail(ctx context.Context, email string) (domain.User, string, error)
	ByID(ctx context.Context, id string) (domain.User, error)
}

// SessionStore persists browser sessions keyed by the hash of the cookie token.
type SessionStore interface {
	CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error
	SessionUserID(ctx context.Context, tokenHash string, now time.Time) (string, error)
	DeleteSession(ctx context.Context, tokenHash string) error
}

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

// Tokens mints opaque secrets and identifiers.
type Tokens interface {
	New() (plain, hash string, err error)
	Hash(plain string) string
	ID() (string, error)
}
