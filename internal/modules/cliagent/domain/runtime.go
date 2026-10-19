// Package domain says what it means to use a vendor's own coding program with a person's own plan.
package domain

import "errors"

// Runtime is a vendor's coding program a person can sign in to with their own plan.
type Runtime string

// The programs people can sign in to. The names are what the API and the screens use.
const (
	Claude  Runtime = "claude"
	ChatGPT Runtime = "chatgpt"
)

var (
	// ErrUnknown means the name is not one of the programs.
	ErrUnknown = errors.New("unknown program")
	// ErrNoSignIn means there is no sign-in in progress to give a code to.
	ErrNoSignIn = errors.New("no sign-in in progress")
	// ErrBadCode means the pasted code cannot be one the vendor gave.
	ErrBadCode = errors.New("that code does not look right")
	// ErrFailed means the sign-in did not finish: refused, timed out or cancelled.
	ErrFailed = errors.New("the sign-in did not finish")
	// ErrUnavailable means the sandbox that runs the program cannot be used right now.
	ErrUnavailable = errors.New("the program cannot be started right now")
)

// Parse turns the name used by the API into a Runtime.
func Parse(name string) (Runtime, error) {
	switch r := Runtime(name); r {
	case Claude, ChatGPT:
		return r, nil
	default:
		return "", ErrUnknown
	}
}

// Prompt is what a person must do in their browser to finish signing in: open the address and,
// for programs that show one, type the code that goes with it.
type Prompt struct {
	URL  string `json:"url"`
	Code string `json:"code,omitempty"`
}

// State is where a person stands with one program.
type State struct {
	// Prompt is set while a sign-in waits for the person.
	Prompt    *Prompt
	Connected bool
}
