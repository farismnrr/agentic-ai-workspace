package domain

import "errors"

// Errors of the task use cases.
var (
	ErrNotFound       = errors.New("task not found")
	ErrBadRequest     = errors.New("write what you need")
	ErrNoProject      = errors.New("project not found")
	ErrBadEffort      = errors.New("that effort level is not available")
	ErrBadMode        = errors.New("task mode must be swift or deep-review")
	ErrNoModel        = errors.New("no model is available; add a provider under Settings, AI models")
	ErrBusy           = errors.New("the agent is still working on this task")
	ErrNeedsAnswer    = errors.New("answer the pending question or approval first")
	ErrNotWaiting     = errors.New("nothing is waiting for that answer")
	ErrWrongState     = errors.New("that is not possible in the task's current state")
	ErrUnknownWork    = errors.New("work session not found")
	ErrWorkClosed     = errors.New("this work session has ended; start a new one with begin_work")
	ErrStaleRevision  = errors.New("work session changed; refresh and try again")
	ErrKeyReused      = errors.New("idempotency key was already used for a different request")
	ErrBadRole        = errors.New("role is not available")
	ErrTooMany        = errors.New("too many items")
	ErrRateLimited    = errors.New("too many requests; try again shortly")
	ErrTooLarge       = errors.New("input is too large")
	ErrAwaitingPerson = errors.New("this plan awaits the person's review; wait or use outcome blocked")
	ErrNoResearch     = errors.New("save sourced research or mark this as a small task before submitting the plan")
	ErrOpenQuestions  = errors.New("wait for the person's answers before submitting the plan")
	ErrTryAgain       = errors.New("the workspace is busy; try again")
	ErrUnverifiedUser = errors.New("sign in with a verified account before approving a plan")
)

// ErrUndoRefused means a change cannot be undone because the file changed again.
var ErrUndoRefused = errors.New("the file changed after this step, so it was left as it is")
