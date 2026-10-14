package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// SaveCode stores a hashed authorization code.
func (s *Store) SaveCode(ctx context.Context, c domain.CodeRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO oauth_codes (hash, grant_id, code_challenge, nonce, expires_at) VALUES (?, ?, ?, ?, ?)`,
		c.Hash, c.GrantID, c.CodeChallenge, c.Nonce, c.ExpiresAt.Unix())
	if err != nil {
		return fmt.Errorf("insert code: %w", err)
	}

	return nil
}

// ConsumeCode atomically marks an unexpired code as used. A replayed code
// returns its grant id together with ErrCredentialReused.
func (s *Store) ConsumeCode(ctx context.Context, hash string, now time.Time) (domain.CodeRecord, error) {
	record := domain.CodeRecord{Hash: hash}

	err := s.db.QueryRowContext(ctx,
		`UPDATE oauth_codes SET consumed_at = ? WHERE hash = ? AND consumed_at IS NULL AND expires_at > ?
		 RETURNING grant_id, code_challenge, nonce`, now.Unix(), hash, now.Unix()).
		Scan(&record.GrantID, &record.CodeChallenge, &record.Nonce)
	if err == nil {
		return record, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return domain.CodeRecord{}, fmt.Errorf("consume code: %w", err)
	}

	err = s.db.QueryRowContext(ctx, `SELECT grant_id FROM oauth_codes WHERE hash = ? AND consumed_at IS NOT NULL`, hash).
		Scan(&record.GrantID)
	if err == nil {
		return record, domain.ErrCredentialReused
	}

	return domain.CodeRecord{}, notFound(err)
}

// SaveToken stores a hashed access or refresh token.
func (s *Store) SaveToken(ctx context.Context, t domain.TokenRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO oauth_tokens (hash, grant_id, kind, expires_at) VALUES (?, ?, ?, ?)`,
		t.Hash, t.GrantID, string(t.Kind), t.ExpiresAt.Unix())
	if err != nil {
		return fmt.Errorf("insert token: %w", err)
	}

	return nil
}

// FindToken loads an unexpired, unconsumed token of the given kind.
func (s *Store) FindToken(
	ctx context.Context, hash string, kind domain.TokenKind, now time.Time,
) (domain.TokenRecord, error) {
	record := domain.TokenRecord{Hash: hash, Kind: kind}

	var expires int64

	err := s.db.QueryRowContext(ctx,
		`SELECT grant_id, expires_at FROM oauth_tokens
		 WHERE hash = ? AND kind = ? AND expires_at > ? AND consumed_at IS NULL`, hash, string(kind), now.Unix()).
		Scan(&record.GrantID, &expires)
	if err != nil {
		return domain.TokenRecord{}, notFound(err)
	}

	record.ExpiresAt = time.Unix(expires, 0)

	return record, nil
}

// ConsumeRefresh atomically rotates out a refresh token; a replay returns its
// grant id together with ErrCredentialReused.
func (s *Store) ConsumeRefresh(ctx context.Context, hash string, now time.Time) (domain.TokenRecord, error) {
	record := domain.TokenRecord{Hash: hash, Kind: domain.TokenRefresh}

	err := s.db.QueryRowContext(ctx,
		`UPDATE oauth_tokens SET consumed_at = ?
		 WHERE hash = ? AND kind = 'refresh' AND consumed_at IS NULL AND expires_at > ? RETURNING grant_id`,
		now.Unix(), hash, now.Unix()).Scan(&record.GrantID)
	if err == nil {
		return record, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return domain.TokenRecord{}, fmt.Errorf("consume refresh token: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		`SELECT grant_id FROM oauth_tokens WHERE hash = ? AND kind = 'refresh' AND consumed_at IS NOT NULL`, hash).
		Scan(&record.GrantID)
	if err == nil {
		return record, domain.ErrCredentialReused
	}

	return domain.TokenRecord{}, notFound(err)
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrCredentialNotFound
	}

	return fmt.Errorf("select credential: %w", err)
}
