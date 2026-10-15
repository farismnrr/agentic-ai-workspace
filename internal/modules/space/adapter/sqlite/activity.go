package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
)

const activityColumns = `id, user_id, space, client, tool, summary, outcome, detail, at, undo, undone_at, ` +
	`project_id, task_id`

type scanner interface {
	Scan(dest ...any) error
}

// AddActivity stores a log entry.
func (s *Store) AddActivity(ctx context.Context, e domain.Activity) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO activity (`+activityColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)`,
		e.ID, e.UserID, e.Space, e.Client, e.Tool, e.Summary, string(e.Outcome), e.Detail, e.At.Unix(), e.Undo,
		e.ProjectID, e.TaskID)
	if err != nil {
		return fmt.Errorf("insert activity: %w", err)
	}

	return nil
}

// ListActivity returns a person's latest entries, newest first.
func (s *Store) ListActivity(ctx context.Context, userID string, limit int) ([]domain.Activity, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+activityColumns+` FROM activity WHERE user_id = ? ORDER BY at DESC, rowid DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("select activity: %w", err)
	}

	defer func() { _ = rows.Close() }()

	entries := []domain.Activity{}

	for rows.Next() {
		entry, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read activity: %w", err)
	}

	return entries, nil
}

// FindActivity loads one entry.
func (s *Store) FindActivity(ctx context.Context, id string) (domain.Activity, error) {
	entry, err := scanActivity(s.db.QueryRowContext(ctx, `SELECT `+activityColumns+` FROM activity WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Activity{}, domain.ErrNoActivity
	}

	return entry, err
}

// MarkUndone records that an entry's change was reversed.
func (s *Store) MarkUndone(ctx context.Context, id string, at time.Time) error {
	if _, err := s.db.ExecContext(ctx, `UPDATE activity SET undone_at = ? WHERE id = ?`, at.Unix(), id); err != nil {
		return fmt.Errorf("mark undone: %w", err)
	}

	return nil
}

func scanActivity(row scanner) (domain.Activity, error) {
	var (
		e       domain.Activity
		outcome string
		at      int64
		undone  sql.NullInt64
	)

	err := row.Scan(&e.ID, &e.UserID, &e.Space, &e.Client, &e.Tool, &e.Summary, &outcome, &e.Detail, &at, &e.Undo, &undone,
		&e.ProjectID, &e.TaskID)
	if err != nil {
		return domain.Activity{}, fmt.Errorf("scan activity: %w", err)
	}

	e.Outcome, e.At = domain.Outcome(outcome), time.Unix(at, 0)
	if undone.Valid {
		moment := time.Unix(undone.Int64, 0)
		e.UndoneAt = &moment
	}

	return e, nil
}
