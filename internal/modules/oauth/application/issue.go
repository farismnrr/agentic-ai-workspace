package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// TokenResponse is the body of a successful token endpoint response.
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	Scope        string
	IDToken      string
	ExpiresIn    int
}

// issue mints an access token, a rotated refresh token and, for openid
// grants, an ID token.
func (s *Service) issue(ctx context.Context, grant domain.Grant, nonce string) (TokenResponse, error) {
	access, err := s.mint(ctx, grant.ID, domain.TokenAccess, accessTokenTTL)
	if err != nil {
		return TokenResponse{}, err
	}

	refresh, err := s.mint(ctx, grant.ID, domain.TokenRefresh, refreshTokenTTL)
	if err != nil {
		return TokenResponse{}, err
	}

	response := TokenResponse{
		AccessToken: access, ExpiresIn: int(accessTokenTTL.Seconds()), RefreshToken: refresh,
		Scope: domain.JoinScopes(grant.Scopes),
	}

	if domain.HasScope(grant.Scopes, domain.ScopeOpenID) {
		response.IDToken, err = s.idToken(ctx, grant, nonce)
		if err != nil {
			return TokenResponse{}, err
		}
	}

	return response, nil
}

func (s *Service) mint(ctx context.Context, grantID string, kind domain.TokenKind, ttl time.Duration) (string, error) {
	plain, hash, err := s.Tokens.New()
	if err != nil {
		return "", fmt.Errorf("new %s token: %w", kind, err)
	}

	record := domain.TokenRecord{Hash: hash, GrantID: grantID, Kind: kind, ExpiresAt: s.Now().Add(ttl)}
	if err := s.Credentials.SaveToken(ctx, record); err != nil {
		return "", fmt.Errorf("save %s token: %w", kind, err)
	}

	return plain, nil
}

func (s *Service) idToken(ctx context.Context, grant domain.Grant, nonce string) (string, error) {
	profile, err := s.Users.Profile(ctx, grant.UserID)
	if err != nil {
		return "", fmt.Errorf("load profile: %w", err)
	}

	claims := IDClaims{
		Subject: grant.UserID, Audience: grant.ClientID, Nonce: nonce,
		IssuedAt: s.Now(), ExpiresAt: s.Now().Add(accessTokenTTL),
	}
	if domain.HasScope(grant.Scopes, domain.ScopeEmail) {
		claims.Email, claims.EmailVerified = profile.Email, true
	}

	if domain.HasScope(grant.Scopes, domain.ScopeProfile) {
		claims.Name = profile.Name
	}

	token, err := s.Signer.Sign(claims)
	if err != nil {
		return "", fmt.Errorf("sign id token: %w", err)
	}

	return token, nil
}
