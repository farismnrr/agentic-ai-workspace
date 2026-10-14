package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// AccessInfo describes a verified access token.
type AccessInfo struct {
	UserID    string
	GrantID   string
	ClientID  string
	Resource  string
	Scopes    []string
	ExpiresAt int64
}

// VerifyAccessToken resolves a bearer token to the grant behind it. Callers
// must still check that Resource is their own (audience validation).
func (s *Service) VerifyAccessToken(ctx context.Context, token string) (AccessInfo, error) {
	now := s.Now()

	record, err := s.Credentials.FindToken(ctx, s.Tokens.Hash(token), domain.TokenAccess, now)
	if errors.Is(err, domain.ErrCredentialNotFound) {
		return AccessInfo{}, domain.ErrInvalidToken
	}

	if err != nil {
		return AccessInfo{}, fmt.Errorf("find token: %w", err)
	}

	grant, err := s.Grants.FindGrant(ctx, record.GrantID)
	if err != nil || !grant.Active() {
		return AccessInfo{}, domain.ErrInvalidToken
	}

	if grant.LastUsedAt == nil || now.Sub(*grant.LastUsedAt) > touchInterval {
		_ = s.Grants.TouchGrant(ctx, grant.ID, now)
	}

	return AccessInfo{
		UserID: grant.UserID, GrantID: grant.ID, ClientID: grant.ClientID, Resource: grant.Resource,
		Scopes: grant.Scopes, ExpiresAt: record.ExpiresAt.Unix(),
	}, nil
}

// UserInfo returns the OpenID Connect claims the token's scopes allow.
func (s *Service) UserInfo(ctx context.Context, token string) (map[string]any, error) {
	info, err := s.VerifyAccessToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if !domain.HasScope(info.Scopes, domain.ScopeOpenID) {
		return nil, domain.ErrInvalidToken
	}

	profile, err := s.Users.Profile(ctx, info.UserID)
	if err != nil {
		return nil, fmt.Errorf("load profile: %w", err)
	}

	claims := map[string]any{"sub": info.UserID}
	if domain.HasScope(info.Scopes, domain.ScopeEmail) {
		claims["email"], claims["email_verified"] = profile.Email, true
	}

	if domain.HasScope(info.Scopes, domain.ScopeProfile) {
		claims["name"] = profile.Name
	}

	return claims, nil
}
