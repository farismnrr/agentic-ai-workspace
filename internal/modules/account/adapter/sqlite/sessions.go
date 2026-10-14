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
	return s.userFor(ctx, domain.ErrNoSession, "select session",
		`SELECT user_id FROM sessions WHERE token_hash = ? AND expires_at > ?`, tokenHash, now.Unix())
}

// userFor runs a query that yields one user id, or missing when there is no row.
func (s *Store) userFor(ctx context.Context, missing error, what, query string, args ...any) (string, error) {
	var userID string

	err := s.db.QueryRowContext(ctx, query, args...).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", missing
	}

	if err != nil {
		return "", fmt.Errorf("%s: %w", what, err)
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

// DeleteSessionsExcept ends all of a person's sessions but the one named by keepHash.
func (s *Store) DeleteSessionsExcept(ctx context.Context, userID, keepHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ? AND token_hash != ?`, userID, keepHash)
	if err != nil {
		return fmt.Errorf("delete sessions: %w", err)
	}

	return nil
}

// CreateReset stores a reset link for a person, replacing earlier ones.
func (s *Store) CreateReset(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM password_resets WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("clear old resets: %w", err)
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO password_resets (token_hash, user_id, expires_at) VALUES (?, ?, ?)`, tokenHash, userID, expiresAt.Unix())
	if err != nil {
		return fmt.Errorf("insert reset: %w", err)
	}

	return nil
}

// ConsumeReset uses a link up once and returns whose it was.
func (s *Store) ConsumeReset(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	return s.userFor(ctx, domain.ErrInvalidResetToken, "consume reset",
		`DELETE FROM password_resets WHERE token_hash = ? AND expires_at > ? RETURNING user_id`, tokenHash, now.Unix())
}
