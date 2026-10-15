package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
)

// PermissionStore persists a person's explicit per-tool choices.
type PermissionStore interface {
	// ToolStates returns the tools the person switched on or off themselves.
	ToolStates(ctx context.Context, userID, space string) (map[string]bool, error)
	SetTool(ctx context.Context, userID, space, tool string, enabled bool, at time.Time) error
	// SkipApprovals reports whether the person lets the assistant use every enabled tool of a space without asking first.
	SkipApprovals(ctx context.Context, userID, space string) (bool, error)
	SetSkipApprovals(ctx context.Context, userID, space string, skip bool, at time.Time) error
}

// ActivityStore persists the activity log.
type ActivityStore interface {
	AddActivity(ctx context.Context, entry domain.Activity) error
	ListActivity(ctx context.Context, userID string, limit int) ([]domain.Activity, error)
	FindActivity(ctx context.Context, id string) (domain.Activity, error)
	MarkUndone(ctx context.Context, id string, at time.Time) error
}

// Undoer reverses a change recorded by one space's tools. A refusal (for
// example because the file changed again) must wrap domain.ErrUndoRefused.
type Undoer interface {
	Undo(ctx context.Context, userID string, change []byte) error
}

// IDs mints identifiers.
type IDs interface {
	ID() (string, error)
}
