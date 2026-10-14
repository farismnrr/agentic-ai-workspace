package domain

import "errors"

// Errors of the task use cases.
var (
	ErrNotFound    = errors.New("task not found")
	ErrBadRequest  = errors.New("write what you need in a few words")
	ErrNoProject   = errors.New("project not found")
	ErrNoModel     = errors.New("no model is available; add a provider under Settings, AI models")
	ErrBusy        = errors.New("the agent is still working on this task")
	ErrNeedsAnswer = errors.New("answer the pending question or approval first")
	ErrNotWaiting  = errors.New("nothing is waiting for that answer")
	ErrWrongState  = errors.New("that is not possible in the task's current state")
)

// ErrUndoRefused means a change cannot be undone because the file changed again.
var ErrUndoRefused = errors.New("the file changed after this step, so it was left as it is")
