package sqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

// SetVerified records the result of a verification, the models found and what the provider reported about them
// (context window, how they think). The previous list is replaced, so a model that went away does not linger.
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

	if err := replaceCatalog(ctx, tx, id, models); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit verification: %w", err)
	}

	return nil
}
