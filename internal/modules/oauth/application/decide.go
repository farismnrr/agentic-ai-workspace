package application

import (
	"context"
	"fmt"
	"net/url"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// Decide records the person's answer to a pending request and returns the URL
// the browser must be sent to: the client's redirect URI with a code or error.
func (s *Service) Decide(ctx context.Context, userID, requestID string, approve bool) (string, error) {
	request, err := s.pending(ctx, requestID, s.Now())
	if err != nil {
		return "", err
	}

	if err = s.Requests.DeleteRequest(ctx, requestID); err != nil {
		return "", fmt.Errorf("delete request: %w", err)
	}

	if !approve {
		return s.redirect(request.RedirectURI, request.State, map[string]string{"error": domain.CodeAccessDenied})
	}

	code, err := s.approve(ctx, userID, request)
	if err != nil {
		return "", err
	}

	return s.redirect(request.RedirectURI, request.State, map[string]string{"code": code})
}

func (s *Service) approve(ctx context.Context, userID string, request domain.AuthorizationRequest) (string, error) {
	client, err := s.Clients.FindClient(ctx, request.ClientID)
	if err != nil {
		return "", fmt.Errorf("find client: %w", err)
	}

	grantID, err := s.Tokens.ID()
	if err != nil {
		return "", fmt.Errorf("new grant id: %w", err)
	}

	grant := domain.Grant{
		ID: grantID, UserID: userID, ClientID: client.ID, ClientName: client.Name, RedirectURI: request.RedirectURI,
		Resource: request.Resource, Scopes: request.Scopes, CreatedAt: s.Now(),
	}
	if err = s.Grants.CreateGrant(ctx, grant); err != nil {
		return "", fmt.Errorf("create grant: %w", err)
	}

	plain, hash, err := s.Tokens.New()
	if err != nil {
		return "", fmt.Errorf("new code: %w", err)
	}

	record := domain.CodeRecord{
		Hash: hash, GrantID: grantID, CodeChallenge: request.CodeChallenge, Nonce: request.Nonce,
		ExpiresAt: s.Now().Add(codeTTL),
	}
	if err := s.Credentials.SaveCode(ctx, record); err != nil {
		return "", fmt.Errorf("save code: %w", err)
	}

	return plain, nil
}

// redirect appends params, the client's state and the RFC 9207 iss to a redirect URI.
func (s *Service) redirect(base, state string, params map[string]string) (string, error) {
	target, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse redirect uri: %w", err)
	}

	query := target.Query()
	for key, value := range params {
		query.Set(key, value)
	}

	if state != "" {
		query.Set("state", state)
	}

	query.Set("iss", s.Issuer)
	target.RawQuery = query.Encode()

	return target.String(), nil
}

// RedirectError renders an OAuth error as a redirect back to the client (RFC 6749 §4.1.2.1).
func (s *Service) RedirectError(oauthErr *domain.Error) (string, error) {
	params := map[string]string{"error": oauthErr.Code}
	if oauthErr.Description != "" {
		params["error_description"] = oauthErr.Description
	}

	return s.redirect(oauthErr.RedirectURI, oauthErr.State, params)
}
