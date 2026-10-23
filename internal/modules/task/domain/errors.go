package domain

import "errors"

// Errors of the task use cases.
var (
	ErrNotFound        = errors.New("task not found")
	ErrBadRequest      = errors.New("write what you need")
	ErrNoProject       = errors.New("project not found")
	ErrBadEffort       = errors.New("that effort level is not available")
	ErrNoModel         = errors.New("no model is available; add a provider under Settings, AI models")
	ErrTeamUnavailable = errors.New("the Orchestrator team is unavailable; check the team setup and try again")
	ErrBusy            = errors.New("the agent is still working on this task")
	ErrNeedsAnswer     = errors.New("answer the pending question or approval first")
	ErrNotWaiting      = errors.New("nothing is waiting for that answer")
	ErrWrongState      = errors.New("that is not possible in the task's current state")
)

// ErrUndoRefused means a change cannot be undone because the file changed again.
var ErrUndoRefused = errors.New("the file changed after this step, so it was left as it is")
