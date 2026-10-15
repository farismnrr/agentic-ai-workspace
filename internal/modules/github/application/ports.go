// Package application holds the GitHub connection use cases.
package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

// Record is a stored connection; the tokens are sealed.
type Record struct {
	AccessExpires  time.Time
	RefreshExpires time.Time
	UserID         string
	AccessSealed   string
	RefreshSealed  string
	Profile        domain.Profile
}

// Store persists one connection per person.
type Store interface {
	Load(ctx context.Context, userID string) (Record, bool, error)
	Save(ctx context.Context, rec Record, at time.Time) error
	Delete(ctx context.Context, userID string) error
}

// Provider is GitHub itself. ErrRejected marks credentials GitHub refuses.
type Provider interface {
	AuthorizeURL(state, challenge string) string
	Exchange(ctx context.Context, code, verifier string) (domain.Tokens, error)
	Refresh(ctx context.Context, refresh string) (domain.Tokens, error)
	Profile(ctx context.Context, access string) (domain.Profile, error)
	Revoke(ctx context.Context, access string) error
}

// Sealer encrypts tokens at rest.
type Sealer interface {
	Encrypt(plain []byte) (string, error)
	Decrypt(sealed string) ([]byte, error)
}

// Secrets mints the random state and PKCE verifier.
type Secrets interface {
	New() (plain, hash string, err error)
}
