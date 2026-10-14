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
	PasswordHash(ctx context.Context, id string) (string, error)
	SetPassword(ctx context.Context, id, passwordHash string) error
	Delete(ctx context.Context, id string) error
}

// Eraser removes what a feature keeps for a person, when their account is deleted.
type Eraser interface {
	Erase(ctx context.Context, userID string) error
}

// SessionStore persists browser sessions keyed by the hash of the cookie token.
type SessionStore interface {
	CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error
	SessionUserID(ctx context.Context, tokenHash string, now time.Time) (string, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	// DeleteSessionsExcept signs a person out everywhere except the session keep (empty keeps none).
	DeleteSessionsExcept(ctx context.Context, userID, keepHash string) error
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
