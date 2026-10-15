// Package sqlite stores GitHub connections.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/application"
)

// Store persists one connection per person.
type Store struct {
	db *sql.DB
}

// NewStore wraps an open database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Load returns the person's connection, if any.
func (s *Store) Load(ctx context.Context, userID string) (application.Record, bool, error) {
	var (
		rec               application.Record
		access, refreshAt int64
	)

	rec.UserID = userID

	err := s.db.QueryRowContext(ctx,
		`SELECT github_id, login, name, access_sealed, refresh_sealed, access_expires, refresh_expires
		 FROM github_connections WHERE user_id = ?`, userID).
		Scan(&rec.Profile.ID, &rec.Profile.Login, &rec.Profile.Name,
			&rec.AccessSealed, &rec.RefreshSealed, &access, &refreshAt)
	if errors.Is(err, sql.ErrNoRows) {
		return application.Record{}, false, nil
	}

	if err != nil {
		return application.Record{}, false, fmt.Errorf("select github connection: %w", err)
	}

	rec.AccessExpires, rec.RefreshExpires = fromUnix(access), fromUnix(refreshAt)

	return rec, true, nil
}

// Save inserts or replaces the person's connection.
func (s *Store) Save(ctx context.Context, rec application.Record, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO github_connections
		   (user_id, github_id, login, name, access_sealed, refresh_sealed, access_expires, refresh_expires, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (user_id) DO UPDATE SET github_id = excluded.github_id, login = excluded.login,
		   name = excluded.name, access_sealed = excluded.access_sealed, refresh_sealed = excluded.refresh_sealed,
		   access_expires = excluded.access_expires, refresh_expires = excluded.refresh_expires`,
		rec.UserID, rec.Profile.ID, rec.Profile.Login, rec.Profile.Name, rec.AccessSealed, rec.RefreshSealed,
		toUnix(rec.AccessExpires), toUnix(rec.RefreshExpires), at.Unix())
	if err != nil {
		return fmt.Errorf("upsert github connection: %w", err)
	}

	return nil
}

// Delete forgets the person's connection.
func (s *Store) Delete(ctx context.Context, userID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM github_connections WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete github connection: %w", err)
	}

	return nil
}

func toUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}

	return t.Unix()
}

func fromUnix(n int64) time.Time {
	if n == 0 {
		return time.Time{}
	}

	return time.Unix(n, 0)
}

var _ application.Store = (*Store)(nil)
