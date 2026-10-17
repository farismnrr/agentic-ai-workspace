package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Writer changes files inside the workspace. Implementations must refuse to
// reach outside the workspace root, just like the read side.
type Writer interface {
	// WriteFile replaces or creates a file atomically, creating missing folders.
	WriteFile(rel string, data []byte) error
	// Remove deletes one file.
	Remove(rel string) error
	// RemoveAll deletes a folder and everything in it.
	RemoveAll(rel string) error
	// Rename moves a file, creating the destination's folders.
	Rename(from, to string) error
}

// Backups keeps the previous content of changed files so a change can be undone.
type Backups interface {
	Save(id string, data []byte) error
	Load(id string) ([]byte, error)
}

// IDs mints identifiers for backups.
type IDs interface {
	ID() (string, error)
}

// TaskFile returns the raw task definitions, or domain.ErrNoTasks when absent.
type TaskFile interface {
	ReadTasks() ([]byte, error)
	WriteTasks(data []byte) error
}

// RunResult is what a finished task produced.
type RunResult struct {
	Output    string
	ExitCode  int
	Duration  time.Duration
	TimedOut  bool
	Truncated bool
}

// Runner executes a task's command inside the workspace.
type Runner interface {
	Run(ctx context.Context, task domain.Task) (RunResult, error)
}
