package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// Windows persists the context window a person chose for each model of a plan.
type Windows struct{ db *sql.DB }

// NewWindows wraps an open database.
func NewWindows(db *sql.DB) *Windows { return &Windows{db: db} }

// Set records the size chosen for one model.
func (w *Windows) Set(ctx context.Context, userID, program, model string, tokens int64) error {
	_, err := w.db.ExecContext(ctx,
		`INSERT INTO plan_windows (user_id, program, model, context_window) VALUES (?, ?, ?, ?)
		 ON CONFLICT (user_id, program, model) DO UPDATE SET context_window = excluded.context_window`,
		userID, program, model, tokens)
	if err != nil {
		return fmt.Errorf("save plan window: %w", err)
	}

	return nil
}

// All lists the sizes chosen for a plan's models, keyed by model.
func (w *Windows) All(ctx context.Context, userID, program string) (map[string]int64, error) {
	rows, err := w.db.QueryContext(ctx,
		`SELECT model, context_window FROM plan_windows WHERE user_id = ? AND program = ?`, userID, program)
	if err != nil {
		return nil, fmt.Errorf("read plan windows: %w", err)
	}

	defer func() { _ = rows.Close() }()

	sizes := map[string]int64{}

	for rows.Next() {
		var (
			model  string
			tokens int64
		)

		if err = rows.Scan(&model, &tokens); err != nil {
			return nil, fmt.Errorf("scan plan window: %w", err)
		}

		sizes[model] = tokens
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read plan windows: %w", err)
	}

	return sizes, nil
}
