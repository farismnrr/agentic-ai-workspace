package domain

import "time"

// TokenKind distinguishes the two bearer credentials a grant can mint.
type TokenKind string

// Token kinds.
const (
	TokenAccess  TokenKind = "access"
	TokenRefresh TokenKind = "refresh"
)

// Grant is a person's consent for one client to use one resource.
type Grant struct {
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
	ID          string
	UserID      string
	ClientID    string
	ClientName  string
	RedirectURI string
	Resource    string
	Scopes      []string
}

// Active reports whether the grant can still mint or accept tokens.
func (g Grant) Active() bool {
	return g.RevokedAt == nil
}

// AuthorizationRequest is a validated request waiting for the person's consent.
type AuthorizationRequest struct {
	ExpiresAt     time.Time
	ID            string
	ClientID      string
	RedirectURI   string
	State         string
	CodeChallenge string
	Nonce         string
	Resource      string
	Scopes        []string
}

// CodeRecord is a stored, hashed authorization code.
type CodeRecord struct {
	ExpiresAt     time.Time
	Hash          string
	GrantID       string
	CodeChallenge string
	Nonce         string
}

// TokenRecord is a stored, hashed access or refresh token.
type TokenRecord struct {
	ExpiresAt time.Time
	Hash      string
	GrantID   string
	Kind      TokenKind
}
