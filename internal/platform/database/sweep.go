package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	day            = 24 * time.Hour
	revokedKeep    = 30 * day
	unusedClient   = 30 * day
	expiredKeep    = day
	sweepStatement = "sweep"
)

// Sweep deletes rows that can never matter again: expired sessions and sign-in
// requests, long-expired codes and tokens, grants revoked a month ago, and
// clients that never completed a consent.
func Sweep(ctx context.Context, db *sql.DB, now time.Time) error {
	statements := []struct {
		query string
		arg   int64
	}{
		{`DELETE FROM sessions WHERE expires_at < ?`, now.Unix()},
		{`DELETE FROM oauth_requests WHERE expires_at < ?`, now.Unix()},
		{`DELETE FROM oauth_codes WHERE expires_at < ?`, now.Add(-expiredKeep).Unix()},
		{`DELETE FROM oauth_tokens WHERE expires_at < ?`, now.Add(-expiredKeep).Unix()},
		{`DELETE FROM oauth_grants WHERE revoked_at IS NOT NULL AND revoked_at < ?`, now.Add(-revokedKeep).Unix()},
		{
			`DELETE FROM oauth_clients WHERE created_at < ? AND id NOT IN (SELECT client_id FROM oauth_grants)`,
			now.Add(-unusedClient).Unix(),
		},
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query, statement.arg); err != nil {
			return fmt.Errorf("%s: %w", sweepStatement, err)
		}
	}

	return nil
}

// StartSweeper runs Sweep now and then every interval until the returned function is called.
func StartSweeper(db *sql.DB, interval time.Duration) func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			_ = Sweep(ctx, db, time.Now())

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}
