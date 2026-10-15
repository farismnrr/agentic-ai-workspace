package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

// replaceCatalog swaps what a provider reported about its models for what was just read.
func replaceCatalog(ctx context.Context, tx *sql.Tx, id string, models []domain.ModelInfo) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM model_catalog WHERE provider_id = ?`, id); err != nil {
		return fmt.Errorf("clear model catalog: %w", err)
	}

	for _, model := range models {
		levels, err := json.Marshal(append([]string{}, model.Thinking.Levels...))
		if err != nil {
			return fmt.Errorf("encode effort levels: %w", err)
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO model_catalog (provider_id, model, context_window, reported, reasoning, can_disable, levels)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id, model.Name, model.ContextWindow, model.Thinking.Reported, model.Thinking.Reasoning,
			model.Thinking.CanDisable, string(levels)); err != nil {
			return fmt.Errorf("save model catalog: %w", err)
		}
	}

	return nil
}

const catalogRows = `SELECT m.provider_id, m.model, m.context_window, m.reported, m.reasoning, m.can_disable, m.levels
	 FROM model_catalog m JOIN providers p ON p.id = m.provider_id WHERE p.user_id = ?`

// loadCatalog reads what a person's providers reported about their models, keyed by provider id and model.
func (s *Store) loadCatalog(ctx context.Context, userID string) (map[string]map[string]domain.ModelInfo, error) {
	rows, err := s.db.QueryContext(ctx, catalogRows, userID)
	if err != nil {
		return nil, fmt.Errorf("select model catalog: %w", err)
	}

	defer func() { _ = rows.Close() }()

	byProvider := map[string]map[string]domain.ModelInfo{}

	for rows.Next() {
		var (
			id     string
			levels string
			info   domain.ModelInfo
		)

		err := rows.Scan(&id, &info.Name, &info.ContextWindow, &info.Thinking.Reported, &info.Thinking.Reasoning,
			&info.Thinking.CanDisable, &levels)
		if err != nil {
			return nil, fmt.Errorf("scan model catalog: %w", err)
		}

		if err := json.Unmarshal([]byte(levels), &info.Thinking.Levels); err != nil {
			return nil, fmt.Errorf("decode effort levels: %w", err)
		}

		if byProvider[id] == nil {
			byProvider[id] = map[string]domain.ModelInfo{}
		}

		byProvider[id][info.Name] = info
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read model catalog: %w", err)
	}

	return byProvider, nil
}
