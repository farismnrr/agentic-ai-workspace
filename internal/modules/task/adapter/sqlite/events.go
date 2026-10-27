package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const eventColumns = `seq, id, at, tool_use_id, body, undo`

// AppendEvent adds an entry to a task's thread and returns it with its id and sequence number.
func (s *Store) AppendEvent(ctx context.Context, taskID string, event domain.Event) (domain.Event, error) {
	event, err := insertEvent(ctx, s.db, s.ids, taskID, event)
	if err != nil {
		return domain.Event{}, err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE tasks SET updated_at = ? WHERE id = ?`, event.Time.Unix(), taskID)

	return event, nil
}

type eventExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func insertEvent(ctx context.Context, exec eventExecutor, ids application.IDs,
	taskID string, event domain.Event,
) (domain.Event, error) {
	id, err := ids.ID()
	if err != nil {
		return domain.Event{}, fmt.Errorf("new event id: %w", err)
	}

	if event.Time.IsZero() {
		event.Time = time.Now()
	}

	event.ID = id

	body, err := json.Marshal(event)
	if err != nil {
		return domain.Event{}, fmt.Errorf("encode event: %w", err)
	}

	err = exec.QueryRowContext(ctx,
		`INSERT INTO task_events (task_id, seq, id, at, kind, tool_use_id, body, undo)
		 VALUES (?, (SELECT COALESCE(MAX(seq), 0) + 1 FROM task_events WHERE task_id = ?), ?, ?, ?, ?, ?, ?)
		 RETURNING seq`,
		taskID, taskID, id, event.Time.Unix(), string(event.Kind), event.ToolUseID, string(body), event.Undo).
		Scan(&event.Seq)
	if err != nil {
		return domain.Event{}, fmt.Errorf("insert event: %w", err)
	}

	return event, nil
}

// EventsAfter returns the thread entries after a sequence number.
func (s *Store) EventsAfter(ctx context.Context, taskID string, seq int64) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+eventColumns+` FROM task_events WHERE task_id = ? AND seq > ? ORDER BY seq`, taskID, seq)
	if err != nil {
		return nil, fmt.Errorf("select events: %w", err)
	}

	defer func() { _ = rows.Close() }()

	events := []domain.Event{}

	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read events: %w", err)
	}

	return events, nil
}

// FindEvent loads one entry of a task's thread.
func (s *Store) FindEvent(ctx context.Context, taskID, eventID string) (domain.Event, error) {
	event, err := scanEvent(s.db.QueryRowContext(ctx,
		`SELECT `+eventColumns+` FROM task_events WHERE task_id = ? AND id = ?`, taskID, eventID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Event{}, domain.ErrNotFound
	}

	return event, err
}

// EventByToolUse finds the approval or question tied to a model tool call.
func (s *Store) EventByToolUse(ctx context.Context, taskID, toolUseID string) (domain.Event, bool, error) {
	event, err := scanEvent(s.db.QueryRowContext(ctx,
		`SELECT `+eventColumns+` FROM task_events WHERE task_id = ? AND tool_use_id = ? AND tool_use_id != ''
		 ORDER BY seq LIMIT 1`, taskID, toolUseID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Event{}, false, nil
	}

	return event, err == nil, err
}

// UpdateEvent rewrites an entry, for example when an approval is answered.
func (s *Store) UpdateEvent(ctx context.Context, taskID string, event domain.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `UPDATE task_events SET body = ?, undo = ? WHERE task_id = ? AND id = ?`,
		string(body), event.Undo, taskID, event.ID)
	if err != nil {
		return fmt.Errorf("update event: %w", err)
	}

	if changed, err := result.RowsAffected(); err == nil && changed == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func scanEvent(row scanner) (domain.Event, error) {
	var (
		event domain.Event
		at    int64
		body  string
		undo  []byte
		tool  string
		seq   int64
	)

	if err := row.Scan(&seq, &event.ID, &at, &tool, &body, &undo); err != nil {
		return domain.Event{}, fmt.Errorf("scan event: %w", err)
	}

	if err := json.Unmarshal([]byte(body), &event); err != nil {
		return domain.Event{}, fmt.Errorf("decode event: %w", err)
	}

	// The stored body was written before the sequence number existed, so the column wins.
	event.Seq, event.Time, event.ToolUseID, event.Undo = seq, time.Unix(at, 0), tool, undo

	return event, nil
}
