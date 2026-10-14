package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// SaveRequest stores a pending authorization request.
func (s *Store) SaveRequest(ctx context.Context, r domain.AuthorizationRequest) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO oauth_requests (id, client_id, redirect_uri, scope, state, code_challenge, nonce, resource, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ClientID, r.RedirectURI, domain.JoinScopes(r.Scopes), r.State, r.CodeChallenge, r.Nonce,
		r.Resource, r.ExpiresAt.Unix())
	if err != nil {
		return fmt.Errorf("insert request: %w", err)
	}

	return nil
}

// FindRequest loads a pending request that has not expired.
func (s *Store) FindRequest(ctx context.Context, id string, now time.Time) (domain.AuthorizationRequest, error) {
	var (
		r       domain.AuthorizationRequest
		scope   string
		expires int64
	)

	err := s.db.QueryRowContext(ctx,
		`SELECT id, client_id, redirect_uri, scope, state, code_challenge, nonce, resource, expires_at
		 FROM oauth_requests WHERE id = ? AND expires_at > ?`, id, now.Unix()).
		Scan(&r.ID, &r.ClientID, &r.RedirectURI, &scope, &r.State, &r.CodeChallenge, &r.Nonce, &r.Resource, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AuthorizationRequest{}, domain.ErrRequestNotFound
	}

	if err != nil {
		return domain.AuthorizationRequest{}, fmt.Errorf("select request: %w", err)
	}

	r.Scopes = domain.SplitScopes(scope)
	r.ExpiresAt = time.Unix(expires, 0)

	return r, nil
}

// DeleteRequest removes a request and sweeps expired ones.
func (s *Store) DeleteRequest(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM oauth_requests WHERE id = ? OR expires_at < ?`, id, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}

	return nil
}
