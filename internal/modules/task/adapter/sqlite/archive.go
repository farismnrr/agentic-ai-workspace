package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	deleteChunkSize      = 500
	placeholderByteCount = 2
)

// Archived returns a person's archived tasks, newest activity first.
func (s *Store) Archived(ctx context.Context, userID string, limit int) ([]domain.Task, error) {
	return s.list(ctx, userID, "", limit, `archived_at IS NOT NULL`)
}

// Archive records the first archive time and leaves the pin intact.
func (s *Store) Archive(ctx context.Context, userID, id string, at time.Time) error {
	if _, err := s.Find(ctx, userID, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET archived_at = COALESCE(archived_at, ?) WHERE id = ? AND user_id = ?`,
		at.Unix(), id, userID)
	if err != nil {
		return fmt.Errorf("archive task: %w", err)
	}
	return nil
}

// Restore clears archive state without changing the pin.
func (s *Store) Restore(ctx context.Context, userID, id string) error {
	if _, err := s.Find(ctx, userID, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET archived_at = NULL WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("restore task: %w", err)
	}
	return nil
}

// DeleteOwned removes task rows and cascaded data in bounded SQL chunks.
func (s *Store) DeleteOwned(ctx context.Context, userID string, ids []string) ([]string, error) {
	deleted := make([]string, 0, len(ids))
	for start := 0; start < len(ids); start += deleteChunkSize {
		end := min(start+deleteChunkSize, len(ids))
		chunk, err := s.deleteChunk(ctx, userID, ids[start:end])
		if err != nil {
			return deleted, err
		}
		deleted = append(deleted, chunk...)
	}
	return deleted, nil
}

func (s *Store) deleteChunk(ctx context.Context, userID string, ids []string) ([]string, error) {
	marks := make([]byte, 0, len(ids)*placeholderByteCount)
	args := make([]any, 0, len(ids)+1)
	for i, id := range ids {
		if i > 0 {
			marks = append(marks, ',')
		}
		marks = append(marks, '?')
		args = append(args, id)
	}
	args = append(args, userID)
	// #nosec G202 -- only the number of bound-value placeholders is generated here.
	query := `DELETE FROM tasks WHERE id IN (` + string(marks) + `) AND user_id = ?`
	rows, err := s.db.QueryContext(ctx, query+` RETURNING id`, args...)
	if err != nil {
		return nil, fmt.Errorf("delete owned tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	deleted := make([]string, 0, len(ids))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return deleted, fmt.Errorf("read deleted task id: %w", err)
		}
		deleted = append(deleted, id)
	}
	if err := rows.Err(); err != nil {
		return deleted, fmt.Errorf("read deleted task ids: %w", err)
	}
	return deleted, nil
}
