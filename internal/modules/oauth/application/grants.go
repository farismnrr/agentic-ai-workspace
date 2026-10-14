package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// ListGrants returns the person's active grants, newest first.
func (s *Service) ListGrants(ctx context.Context, userID string) ([]domain.Grant, error) {
	grants, err := s.Grants.ListGrants(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list grants: %w", err)
	}

	return grants, nil
}

// RevokeGrant removes one of the person's grants; it takes effect immediately.
func (s *Service) RevokeGrant(ctx context.Context, userID, grantID string) error {
	grant, err := s.Grants.FindGrant(ctx, grantID)
	if err != nil {
		return fmt.Errorf("find grant: %w", err)
	}

	if grant.UserID != userID {
		return domain.ErrGrantNotFound
	}

	if err := s.Grants.RevokeGrant(ctx, grantID, s.Now()); err != nil {
		return fmt.Errorf("revoke grant: %w", err)
	}

	return nil
}

// RevokeToken implements RFC 7009: revoking either token kind revokes the grant.
// Unknown tokens are ignored, as the RFC requires.
func (s *Service) RevokeToken(ctx context.Context, token string) error {
	hash, now := s.Tokens.Hash(token), s.Now()

	for _, kind := range []domain.TokenKind{domain.TokenRefresh, domain.TokenAccess} {
		record, err := s.Credentials.FindToken(ctx, hash, kind, now)
		if errors.Is(err, domain.ErrCredentialNotFound) {
			continue
		}

		if err != nil {
			return fmt.Errorf("find token: %w", err)
		}

		if err := s.Grants.RevokeGrant(ctx, record.GrantID, now); err != nil {
			return fmt.Errorf("revoke grant: %w", err)
		}

		return nil
	}

	return nil
}

// ResourceName returns the display name of a resource, falling back to its URI.
func (s *Service) ResourceName(uri string) string {
	if resource, ok := s.Resources.Find(uri); ok && resource.Name != "" {
		return resource.Name
	}

	return uri
}
