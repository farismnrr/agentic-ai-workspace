package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

const columns = `id, user_id, kind, name, base_url, key_hint, verified, enabled, models, created_at, key_sealed`

// Store persists providers in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore wraps an open database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type scanner interface {
	Scan(dest ...any) error
}

// Create inserts a provider with its sealed key.
func (s *Store) Create(ctx context.Context, p domain.Provider, sealedKey string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO providers
		   (id, user_id, kind, name, base_url, key_sealed, key_hint, verified, enabled, models, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 0, 1, '[]', ?)`,
		p.ID, p.UserID, string(p.Kind), p.Name, p.BaseURL, sealedKey, p.KeyHint, p.CreatedAt.Unix())
	if err != nil {
		return fmt.Errorf("insert provider: %w", err)
	}

	return nil
}

// List returns a person's providers, oldest first.
func (s *Store) List(ctx context.Context, userID string) ([]domain.Provider, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+columns+` FROM providers WHERE user_id = ? ORDER BY created_at, rowid`, userID)
	if err != nil {
		return nil, fmt.Errorf("select providers: %w", err)
	}

	defer func() { _ = rows.Close() }()

	providers := []domain.Provider{}

	for rows.Next() {
		provider, _, err := scan(rows)
		if err != nil {
			return nil, err
		}

		providers = append(providers, provider)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read providers: %w", err)
	}

	return providers, nil
}

// Find loads one provider and its sealed key.
func (s *Store) Find(ctx context.Context, userID, id string) (domain.Provider, string, error) {
	provider, sealed, err := scan(s.db.QueryRowContext(ctx,
		`SELECT `+columns+` FROM providers WHERE user_id = ? AND id = ?`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Provider{}, "", domain.ErrNotFound
	}

	return provider, sealed, err
}

// SetVerified records the result of a verification and the models found.
func (s *Store) SetVerified(ctx context.Context, userID, id string, verified bool, models []string) error {
	encoded, err := json.Marshal(append([]string{}, models...))
	if err != nil {
		return fmt.Errorf("encode models: %w", err)
	}

	return s.exec(ctx, `UPDATE providers SET verified = ?, models = ? WHERE user_id = ? AND id = ?`,
		verified, string(encoded), userID, id)
}

// SetEnabled switches a provider on or off.
func (s *Store) SetEnabled(ctx context.Context, userID, id string, enabled bool) error {
	return s.exec(ctx, `UPDATE providers SET enabled = ? WHERE user_id = ? AND id = ?`, enabled, userID, id)
}

// Delete removes a provider.
func (s *Store) Delete(ctx context.Context, userID, id string) error {
	return s.exec(ctx, `DELETE FROM providers WHERE user_id = ? AND id = ?`, userID, id)
}

func (s *Store) exec(ctx context.Context, query string, args ...any) error {
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update providers: %w", err)
	}

	if changed, err := result.RowsAffected(); err == nil && changed == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func scan(row scanner) (domain.Provider, string, error) {
	var (
		p       domain.Provider
		kind    string
		models  string
		created int64
		sealed  string
	)

	err := row.Scan(&p.ID, &p.UserID, &kind, &p.Name, &p.BaseURL, &p.KeyHint, &p.Verified, &p.Enabled, &models,
		&created, &sealed)
	if err != nil {
		return domain.Provider{}, "", fmt.Errorf("scan provider: %w", err)
	}

	p.Kind, p.CreatedAt = domain.Kind(kind), time.Unix(created, 0)
	if err := json.Unmarshal([]byte(models), &p.Models); err != nil {
		return domain.Provider{}, "", fmt.Errorf("decode models: %w", err)
	}

	return p, sealed, nil
}
