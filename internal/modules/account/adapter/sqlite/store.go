package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
)

// Store persists users and sessions in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore wraps an open database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a user; a duplicate email returns domain.ErrEmailTaken.
func (s *Store) Create(ctx context.Context, user domain.User, passwordHash string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, name, email, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		user.ID, user.Name, user.Email, passwordHash, user.Role, user.CreatedAt.Unix())
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return domain.ErrEmailTaken
	}

	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

// ByEmail returns the user and password hash for an email.
func (s *Store) ByEmail(ctx context.Context, email string) (domain.User, string, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, email, role, created_at, password_hash FROM users WHERE email = ?`, email)

	return scanUser(row)
}

// ByID returns a user by identifier.
func (s *Store) ByID(ctx context.Context, id string) (domain.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, email, role, created_at, password_hash FROM users WHERE id = ?`, id)
	user, _, err := scanUser(row)

	return user, err
}

func scanUser(row *sql.Row) (domain.User, string, error) {
	var (
		user    domain.User
		created int64
		hash    string
	)

	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Role, &created, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, "", domain.ErrUserNotFound
	}

	if err != nil {
		return domain.User{}, "", fmt.Errorf("scan user: %w", err)
	}

	user.CreatedAt = time.Unix(created, 0)

	return user, hash, nil
}

// PasswordHash returns the stored hash of a person's password.
func (s *Store) PasswordHash(ctx context.Context, id string) (string, error) {
	var hash string

	err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = ?`, id).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrUserNotFound
	}

	if err != nil {
		return "", fmt.Errorf("select password hash: %w", err)
	}

	return hash, nil
}

// Delete removes the account; sessions and other rows owned by it go with it.
func (s *Store) Delete(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

// SetPassword replaces a person's password hash.
func (s *Store) SetPassword(ctx context.Context, id, passwordHash string) error {
	if _, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	return nil
}
