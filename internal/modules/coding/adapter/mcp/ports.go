package mcp

import "context"

// Outcome is how a tool call ended, as recorded in the activity log.
type Outcome string

// Call outcomes.
const (
	OutcomeOK        Outcome = "ok"
	OutcomeFailed    Outcome = "failed"
	OutcomeOff       Outcome = "off"
	OutcomeForbidden Outcome = "forbidden"
)

// Gate says whether a person has switched a tool on.
type Gate interface {
	Allowed(ctx context.Context, userID, tool string) (bool, error)
}

// Entry is one tool call for the activity log. Undo, when set, is the opaque
// description of how to reverse the call.
type Entry struct {
	UserID string
	Client string
	// Project is where the call worked, so an undo can find it again.
	Project string
	Tool    string
	Summary string
	Detail  string
	Outcome Outcome
	Undo    []byte
}

// Recorder writes the activity log.
type Recorder interface {
	Record(ctx context.Context, entry Entry) error
}
