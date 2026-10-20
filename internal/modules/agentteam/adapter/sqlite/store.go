package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/domain"
)

// Store persists agent teams in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore wraps an open database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Load returns a person's saved team; found is false when they never saved one.
func (s *Store) Load(ctx context.Context, userID string) (domain.Team, bool, error) {
	var team domain.Team

	err := s.db.QueryRowContext(ctx, `SELECT use_by_default FROM agent_teams WHERE user_id = ?`, userID).
		Scan(&team.UseByDefault)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Team{}, false, nil
	}

	if err != nil {
		return domain.Team{}, false, fmt.Errorf("select agent team: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT key, name, purpose, prompt, provider_id, model, effort, tier, access, lead, enabled
		 FROM agent_roles WHERE user_id = ? ORDER BY position`, userID)
	if err != nil {
		return domain.Team{}, false, fmt.Errorf("select agent roles: %w", err)
	}

	defer func() { _ = rows.Close() }()

	team.Roles = []domain.Role{}

	for rows.Next() {
		var (
			role   domain.Role
			access string
		)

		if err := rows.Scan(&role.Key, &role.Name, &role.Purpose, &role.Prompt, &role.ProviderID, &role.Model,
			&role.Effort, &role.Tier, &access, &role.Lead, &role.Enabled); err != nil {
			return domain.Team{}, false, fmt.Errorf("scan agent role: %w", err)
		}

		if err := json.Unmarshal([]byte(access), &role.Access); err != nil {
			return domain.Team{}, false, fmt.Errorf("decode access: %w", err)
		}

		team.Roles = append(team.Roles, role)
	}

	if err := rows.Err(); err != nil {
		return domain.Team{}, false, fmt.Errorf("read agent roles: %w", err)
	}

	return team, true, nil
}

// Save replaces a person's team in one transaction.
func (s *Store) Save(ctx context.Context, userID string, team domain.Team, at time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_roles WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete agent roles: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO agent_teams (user_id, use_by_default, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT (user_id) DO UPDATE SET use_by_default = excluded.use_by_default, updated_at = excluded.updated_at`,
		userID, team.UseByDefault, at.Unix()); err != nil {
		return fmt.Errorf("upsert agent team: %w", err)
	}

	for position, role := range team.Roles {
		access, err := json.Marshal(append([]string{}, role.Access...))
		if err != nil {
			return fmt.Errorf("encode access: %w", err)
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO agent_roles (user_id, key, name, purpose, prompt, provider_id, model, effort, tier, access,
			   lead, enabled, position) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, role.Key, role.Name, role.Purpose, role.Prompt, role.ProviderID, role.Model, role.Effort,
			role.Tier, string(access), role.Lead, role.Enabled, position); err != nil {
			return fmt.Errorf("insert agent role: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// Clear forgets a person's team.
func (s *Store) Clear(ctx context.Context, userID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM agent_teams WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete agent team: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `DELETE FROM agent_roles WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete agent roles: %w", err)
	}

	return nil
}
