package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

// SetVerified records the result of a verification, the models found and the context windows the provider
// reported for them. The previous list and sizes are replaced, so a model that went away does not linger.
func (s *Store) SetVerified(ctx context.Context, userID, id string, verified bool, models []domain.ModelInfo) error {
	names := make([]string, 0, len(models))
	for _, model := range models {
		names = append(names, model.Name)
	}

	encoded, err := json.Marshal(names)
	if err != nil {
		return fmt.Errorf("encode models: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin verification: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `UPDATE providers SET verified = ?, models = ? WHERE user_id = ? AND id = ?`,
		verified, string(encoded), userID, id)
	if err != nil {
		return fmt.Errorf("update provider: %w", err)
	}

	if err := notFoundIfNone(result); err != nil {
		return err
	}

	if err := replaceWindows(ctx, tx, id, models); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit verification: %w", err)
	}

	return nil
}

// replaceWindows swaps the sizes a provider reported for the ones just read.
func replaceWindows(ctx context.Context, tx *sql.Tx, id string, models []domain.ModelInfo) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM model_windows WHERE provider_id = ?`, id); err != nil {
		return fmt.Errorf("clear reported windows: %w", err)
	}

	for _, model := range models {
		if model.ContextWindow <= 0 {
			continue
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO model_windows (provider_id, model, context_window) VALUES (?, ?, ?)`,
			id, model.Name, model.ContextWindow); err != nil {
			return fmt.Errorf("save reported window: %w", err)
		}
	}

	return nil
}
