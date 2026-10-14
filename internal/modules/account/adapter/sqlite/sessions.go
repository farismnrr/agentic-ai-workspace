package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
)

// CreateSession stores a session by token hash.
func (s *Store) CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		tokenHash, userID, expiresAt.Unix())
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

// SessionUserID resolves a non-expired session to its user id.
func (s *Store) SessionUserID(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	var userID string

	err := s.db.QueryRowContext(ctx,
		`SELECT user_id FROM sessions WHERE token_hash = ? AND expires_at > ?`,
		tokenHash, now.Unix()).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNoSession
	}

	if err != nil {
		return "", fmt.Errorf("select session: %w", err)
	}

	return userID, nil
}

// DeleteSession removes a session by token hash.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
