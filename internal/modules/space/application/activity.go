package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
)

// ActivityLog records tool calls and lets a person undo the reversible ones.
type ActivityLog struct {
	store   ActivityStore
	ids     IDs
	undoers map[string]Undoer
	now     func() time.Time
}

// NewActivityLog wires the log; undoers are keyed by space slug.
func NewActivityLog(store ActivityStore, ids IDs, undoers map[string]Undoer, now func() time.Time) *ActivityLog {
	return &ActivityLog{store: store, ids: ids, undoers: undoers, now: now}
}

// Record stores an entry and returns its id.
func (a *ActivityLog) Record(ctx context.Context, entry domain.Activity) (string, error) {
	id, err := a.ids.ID()
	if err != nil {
		return "", fmt.Errorf("new activity id: %w", err)
	}

	entry.ID, entry.At = id, a.now()
	if err := a.store.AddActivity(ctx, entry); err != nil {
		return "", fmt.Errorf("save activity: %w", err)
	}

	return id, nil
}

// List returns a person's latest entries, newest first.
func (a *ActivityLog) List(ctx context.Context, userID string, limit int) ([]domain.Activity, error) {
	entries, err := a.store.ListActivity(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list activity: %w", err)
	}

	return entries, nil
}

// Undo reverses one of the person's entries.
func (a *ActivityLog) Undo(ctx context.Context, userID, id string) error {
	entry, err := a.store.FindActivity(ctx, id)
	if err != nil || entry.UserID != userID {
		return domain.ErrNoActivity
	}

	if entry.UndoneAt != nil {
		return domain.ErrAlreadyUndone
	}

	undoer, ok := a.undoers[entry.Space]
	if !ok || !entry.CanUndo() {
		return domain.ErrNotUndoable
	}

	if err := undoer.Undo(ctx, entry.Undo); err != nil {
		return errors.Join(err)
	}

	if err := a.store.MarkUndone(ctx, id, a.now()); err != nil {
		return fmt.Errorf("mark undone: %w", err)
	}

	return nil
}
