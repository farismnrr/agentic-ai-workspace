package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// ExchangeParams are the token request parameters for grant_type=authorization_code.
type ExchangeParams struct {
	Code         string
	RedirectURI  string
	ClientID     string
	ClientSecret string
	CodeVerifier string
	Resource     string
}

// ExchangeCode trades an authorization code for tokens.
func (s *Service) ExchangeCode(ctx context.Context, p ExchangeParams) (TokenResponse, error) {
	if err := s.authenticateClient(ctx, p.ClientID, p.ClientSecret); err != nil {
		return TokenResponse{}, err
	}

	code, err := s.Credentials.ConsumeCode(ctx, s.Tokens.Hash(p.Code), s.Now())
	if errors.Is(err, domain.ErrCredentialReused) {
		// A replayed code means it may have leaked: kill everything it produced (RFC 6749 §4.1.2).
		_ = s.Grants.RevokeGrant(ctx, code.GrantID, s.Now())
		return TokenResponse{}, domain.NewError(domain.CodeInvalidGrant, "authorization code already used")
	}

	if errors.Is(err, domain.ErrCredentialNotFound) {
		return TokenResponse{}, domain.NewError(domain.CodeInvalidGrant, "authorization code is invalid or expired")
	}

	if err != nil {
		return TokenResponse{}, fmt.Errorf("consume code: %w", err)
	}

	grant, err := s.activeGrant(ctx, code.GrantID)
	if err != nil {
		return TokenResponse{}, err
	}

	if mismatch := checkExchange(grant, code, p); mismatch != nil {
		return TokenResponse{}, mismatch
	}

	return s.issue(ctx, grant, code.Nonce)
}

func checkExchange(grant domain.Grant, code domain.CodeRecord, p ExchangeParams) *domain.Error {
	switch {
	case grant.ClientID != p.ClientID:
		return domain.NewError(domain.CodeInvalidGrant, "code was issued to a different client")
	case grant.RedirectURI != p.RedirectURI:
		return domain.NewError(domain.CodeInvalidGrant, "redirect_uri does not match the authorization request")
	case !domain.VerifyS256(p.CodeVerifier, code.CodeChallenge):
		return domain.NewError(domain.CodeInvalidGrant, "PKCE verification failed")
	case p.Resource != "" && p.Resource != grant.Resource:
		return domain.NewError(domain.CodeInvalidTarget, "resource does not match the authorization request")
	}

	return nil
}

func (s *Service) activeGrant(ctx context.Context, id string) (domain.Grant, error) {
	grant, err := s.Grants.FindGrant(ctx, id)
	if errors.Is(err, domain.ErrGrantNotFound) {
		return domain.Grant{}, domain.NewError(domain.CodeInvalidGrant, "grant no longer exists")
	}

	if err != nil {
		return domain.Grant{}, fmt.Errorf("find grant: %w", err)
	}

	if !grant.Active() {
		return domain.Grant{}, domain.NewError(domain.CodeInvalidGrant, "access was revoked")
	}

	return grant, nil
}
