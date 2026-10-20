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
	// Task is the thread of the app's conversation this call was written into, when it was.
	Task    string
	Outcome Outcome
	Undo    []byte
}

// Recorder writes the activity log.
type Recorder interface {
	Record(ctx context.Context, entry Entry) error
}

// NoteKind is what a step of the app's conversation is.
type NoteKind string

// Kinds of step.
const (
	NoteUser      NoteKind = "user"
	NoteAssistant NoteKind = "assistant"
	NoteAction    NoteKind = "action"
)

// RunNote is a command the app ran, so the thread can show how it ended.
type RunNote struct {
	Name     string
	ExitCode int
	TimedOut bool
}

// Note is one step of the app's conversation, for the person to read later in the app.
type Note struct {
	Run       *RunNote
	Kind      NoteKind
	Text      string
	Technical string
}

// TaskLog writes the app's conversation into a thread the person can open. It is optional.
type TaskLog interface {
	// Log adds the note to the app's current thread in the project and returns the thread's id.
	Log(ctx context.Context, userID, client, projectID string, note Note) (string, error)
	// Begin writes a step that is under way and shows the thread as working until End.
	Begin(ctx context.Context, userID, client, projectID string, note Note) (string, error)
	// End says the call that Begin started is over.
	End(ctx context.Context, taskID string)
}
