package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/domain"
)

// Store persists preferences in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore wraps an open database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Load returns a person's preferences, or empty ones when none were saved.
func (s *Store) Load(ctx context.Context, userID string) (domain.Preferences, error) {
	var (
		p     domain.Preferences
		rules string
	)

	err := s.db.QueryRowContext(ctx,
		`SELECT default_provider_id, default_model, default_effort, persona_name, talk_style, language,
		   about, rules FROM preferences WHERE user_id = ?`, userID).
		Scan(&p.DefaultProviderID, &p.DefaultModel, &p.DefaultEffort, &p.PersonaName, &p.TalkStyle, &p.Language,
			&p.About, &rules)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Preferences{Rules: []domain.Rule{}}, nil
	}

	if err != nil {
		return domain.Preferences{}, fmt.Errorf("select preferences: %w", err)
	}

	if err := json.Unmarshal([]byte(rules), &p.Rules); err != nil {
		return domain.Preferences{}, fmt.Errorf("decode rules: %w", err)
	}

	return p, nil
}

// Save inserts or replaces a person's preferences.
func (s *Store) Save(ctx context.Context, userID string, p domain.Preferences, at time.Time) error {
	rules, err := json.Marshal(append([]domain.Rule{}, p.Rules...))
	if err != nil {
		return fmt.Errorf("encode rules: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO preferences (user_id, default_provider_id, default_model, default_effort, persona_name,
		   talk_style, language, about, rules, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (user_id) DO UPDATE SET default_provider_id = excluded.default_provider_id,
		   default_model = excluded.default_model, default_effort = excluded.default_effort,
		   persona_name = excluded.persona_name, talk_style = excluded.talk_style, language = excluded.language,
		   about = excluded.about, rules = excluded.rules, updated_at = excluded.updated_at`,
		userID, p.DefaultProviderID, p.DefaultModel, p.DefaultEffort, p.PersonaName, p.TalkStyle, p.Language,
		p.About, string(rules), at.Unix())
	if err != nil {
		return fmt.Errorf("upsert preferences: %w", err)
	}

	return nil
}
