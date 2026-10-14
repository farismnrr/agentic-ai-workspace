package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// RefreshParams are the token request parameters for grant_type=refresh_token.
type RefreshParams struct {
	RefreshToken string
	ClientID     string
	ClientSecret string
	Resource     string
}

// Refresh rotates a refresh token and mints a new token pair. The scope of a
// refreshed token is always the scope originally granted.
func (s *Service) Refresh(ctx context.Context, p RefreshParams) (TokenResponse, error) {
	if err := s.authenticateClient(ctx, p.ClientID, p.ClientSecret); err != nil {
		return TokenResponse{}, err
	}

	record, err := s.Credentials.ConsumeRefresh(ctx, s.Tokens.Hash(p.RefreshToken), s.Now())
	if errors.Is(err, domain.ErrCredentialReused) {
		// Reuse of a rotated token signals theft: revoke the whole grant (OAuth 2.1 §4.3.1).
		_ = s.Grants.RevokeGrant(ctx, record.GrantID, s.Now())
		return TokenResponse{}, domain.NewError(domain.CodeInvalidGrant, "refresh token already used")
	}

	if errors.Is(err, domain.ErrCredentialNotFound) {
		return TokenResponse{}, domain.NewError(domain.CodeInvalidGrant, "refresh token is invalid or expired")
	}

	if err != nil {
		return TokenResponse{}, fmt.Errorf("consume refresh token: %w", err)
	}

	grant, err := s.activeGrant(ctx, record.GrantID)
	if err != nil {
		return TokenResponse{}, err
	}

	if grant.ClientID != p.ClientID {
		return TokenResponse{}, domain.NewError(domain.CodeInvalidGrant, "refresh token was issued to a different client")
	}

	if p.Resource != "" && p.Resource != grant.Resource {
		return TokenResponse{}, domain.NewError(domain.CodeInvalidTarget, "resource does not match the grant")
	}

	return s.issue(ctx, grant, "")
}
