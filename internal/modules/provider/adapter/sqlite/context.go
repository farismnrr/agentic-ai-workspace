package sqlite

import (
	"context"
	"fmt"
)

// SetContextWindow records the context window a person chose for one model.
func (s *Store) SetContextWindow(ctx context.Context, userID, id, model string, tokens int64) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO model_settings (provider_id, model, context_window)
		 SELECT id, ?, ? FROM providers WHERE user_id = ? AND id = ?
		 ON CONFLICT (provider_id, model) DO UPDATE SET context_window = excluded.context_window`,
		model, tokens, userID, id)
	if err != nil {
		return fmt.Errorf("save context window: %w", err)
	}

	return notFoundIfNone(result)
}

// loadWindows fills in the chosen context windows of the given providers.
func (s *Store) loadWindows(ctx context.Context, userID string, byID map[string]map[string]int64) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.provider_id, m.model, m.context_window FROM model_settings m
		 JOIN providers p ON p.id = m.provider_id WHERE p.user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("select context windows: %w", err)
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			id, model string
			tokens    int64
		)

		if err := rows.Scan(&id, &model, &tokens); err != nil {
			return fmt.Errorf("scan context window: %w", err)
		}

		byID[id] = withWindow(byID[id], model, tokens)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("read context windows: %w", err)
	}

	return nil
}

func withWindow(windows map[string]int64, model string, tokens int64) map[string]int64 {
	if windows == nil {
		windows = map[string]int64{}
	}

	windows[model] = tokens

	return windows
}
