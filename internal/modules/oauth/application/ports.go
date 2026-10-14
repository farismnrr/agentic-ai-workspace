package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// ClientStore persists known clients.
type ClientStore interface {
	SaveClient(ctx context.Context, client domain.Client) error
	FindClient(ctx context.Context, id string) (domain.Client, error)
}

// RequestStore persists authorization requests awaiting consent.
type RequestStore interface {
	SaveRequest(ctx context.Context, request domain.AuthorizationRequest) error
	FindRequest(ctx context.Context, id string, now time.Time) (domain.AuthorizationRequest, error)
	DeleteRequest(ctx context.Context, id string) error
}

// GrantStore persists consent grants.
type GrantStore interface {
	CreateGrant(ctx context.Context, grant domain.Grant) error
	FindGrant(ctx context.Context, id string) (domain.Grant, error)
	ListGrants(ctx context.Context, userID string) ([]domain.Grant, error)
	RevokeGrant(ctx context.Context, grantID string, now time.Time) error
	TouchGrant(ctx context.Context, grantID string, now time.Time) error
}

// CredentialStore persists hashed authorization codes and tokens.
type CredentialStore interface {
	SaveCode(ctx context.Context, code domain.CodeRecord) error
	// ConsumeCode atomically marks a code used. It returns ErrCredentialNotFound
	// for unknown or expired codes and ErrCredentialReused for a replayed one.
	ConsumeCode(ctx context.Context, hash string, now time.Time) (domain.CodeRecord, error)
	SaveToken(ctx context.Context, token domain.TokenRecord) error
	FindToken(ctx context.Context, hash string, kind domain.TokenKind, now time.Time) (domain.TokenRecord, error)
	// ConsumeRefresh atomically rotates out a refresh token, with the same errors as ConsumeCode.
	ConsumeRefresh(ctx context.Context, hash string, now time.Time) (domain.TokenRecord, error)
}

// MetadataFetcher loads and validates a Client ID Metadata Document.
type MetadataFetcher interface {
	Fetch(ctx context.Context, clientID string) (domain.Client, error)
}

// IDTokenSigner signs OpenID Connect ID tokens.
type IDTokenSigner interface {
	Sign(claims IDClaims) (string, error)
}

// IDClaims are the ID token claims the use cases supply.
type IDClaims struct {
	IssuedAt      time.Time
	ExpiresAt     time.Time
	Subject       string
	Audience      string
	Email         string
	Name          string
	Nonce         string
	EmailVerified bool
}

// Profile is what the application needs to know about a person.
type Profile struct {
	ID    string
	Name  string
	Email string
}

// UserDirectory looks people up by id.
type UserDirectory interface {
	Profile(ctx context.Context, userID string) (Profile, error)
}

// Resource is a protected resource (an MCP space) tokens can be issued for.
type Resource struct {
	URI    string
	Name   string
	Scopes []domain.Scope
}

// ResourceCatalog lists the protected resources this server hosts.
type ResourceCatalog interface {
	Find(uri string) (Resource, bool)
	All() []Resource
}

// Tokens mints opaque secrets and identifiers.
type Tokens interface {
	New() (plain, hash string, err error)
	Hash(plain string) string
	ID() (string, error)
}
