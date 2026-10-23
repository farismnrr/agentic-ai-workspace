package domain

import (
	"errors"
	"time"
)

// Outcome says how a tool call ended.
type Outcome string

// Call outcomes.
const (
	OutcomeOK        Outcome = "ok"
	OutcomeFailed    Outcome = "failed"
	OutcomeOff       Outcome = "off"
	OutcomeForbidden Outcome = "forbidden"
)

// Errors of the permission and activity use cases.
var (
	ErrUnknownSpace         = errors.New("unknown space")
	ErrUnknownTool          = errors.New("unknown tool")
	ErrApprovalsUnsupported = errors.New("space does not support skipped approvals")
	ErrNoActivity           = errors.New("activity entry not found")
	ErrNotUndoable          = errors.New("this entry cannot be undone")
	ErrAlreadyUndone        = errors.New("this entry was already undone")
	ErrUndoRefused          = errors.New("undo refused")
)

// Activity is one recorded tool call. Undo is an opaque blob only the space's
// own module understands.
type Activity struct {
	At       time.Time
	UndoneAt *time.Time
	ID       string
	UserID   string
	Space    string
	Client   string
	// ProjectID and TaskID say where the call happened; either may be empty.
	ProjectID string
	TaskID    string
	Tool      string
	Summary   string
	Detail    string
	Outcome   Outcome
	Undo      []byte
}

// CanUndo reports whether the entry still has something to reverse.
func (a Activity) CanUndo() bool {
	return a.Outcome == OutcomeOK && len(a.Undo) > 0 && a.UndoneAt == nil
}
