package application

import (
	"context"
	"encoding/json"
)

// CLIEventKind is what a vendor's coding program reported.
type CLIEventKind int

// The things a program reports while it works.
const (
	// CLIText is a finished piece of the program's own words.
	CLIText CLIEventKind = iota + 1
	// CLIDelta is a few words of a reply still being written.
	CLIDelta
	// CLIChange is a step that changed files.
	CLIChange
	// CLICommand is a step that ran a command.
	CLICommand
)

// CLIEvent is one thing the program reported, already in plain words.
type CLIEvent struct {
	// Text is the program's words, or a plain sentence describing the step.
	Text string
	// Technical is the exact command or file list behind a step.
	Technical string
	Files     []string
	ExitCode  int
	Kind      CLIEventKind
	// TimedOut is true when a command was stopped for taking too long.
	TimedOut bool
}

// CLIRequest is one turn handed to a vendor's coding program.
type CLIRequest struct {
	// Bridge, when set, gives the program tools the server runs for it, such as handing a job to a teammate.
	Bridge    *CLIBridge
	UserID    string
	ProjectID string
	// Program is which one: "claude" or "chatgpt".
	Program string
	Prompt  string
	// Model and Effort are what the person chose; the program decides what they mean for it.
	Model, Effort string
	// Session continues the conversation the program kept for this task; empty starts a new one.
	Session string
	// Tools, when Limited, are the project tools a teammate's role may use; the program is held to them.
	Tools []string
	// Limited says Tools is a limit, even an empty one.
	Limited bool
}

// CLITool is a tool the server runs for the program.
type CLITool struct {
	Name        string
	Description string
	// Schema is the JSON Schema of its arguments.
	Schema json.RawMessage
}

// CLIBridge is the tools the server runs for a program, and how.
type CLIBridge struct {
	// Call runs a tool the program asked for; failed says the call did not work.
	Call  func(ctx context.Context, name string, input json.RawMessage) (text string, failed bool)
	Tools []CLITool
}

// CLIResult is how a turn ended.
type CLIResult struct {
	Session string
	// Summary is the program's closing words.
	Summary string
	// Reason, when set, says in plain words why the turn failed.
	Reason string
}

// CLIRunner runs a person's own plan through the vendor's coding program, in the project's folder.
type CLIRunner interface {
	// Ready says whether the person is signed in to the program.
	Ready(ctx context.Context, userID, program string) (bool, error)
	// Run does one turn and reports each step to emit as it happens.
	Run(ctx context.Context, req CLIRequest, emit func(CLIEvent)) (CLIResult, error)
}

// CLISessions remembers the conversation a program keeps for each task.
type CLISessions interface {
	Session(ctx context.Context, taskID string) (string, error)
	SaveSession(ctx context.Context, taskID, session string) error
}
