package application

import (
	"context"
	"io"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Writer changes files inside the workspace. Implementations must refuse to
// reach outside the workspace root, just like the read side.
type Writer interface {
	// Create writes a new file from a stream, refuses overwrite, and removes partial data on error.
	Create(rel string, src io.Reader) error
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

// RunResult is what a finished task produced. Effect paths describe only the
// project workspace snapshot; they do not certify effects outside that tree.
type RunResult struct {
	Output       string
	Command      string
	EffectPaths  []string
	ExitCode     int
	Duration     time.Duration
	TimedOut     bool
	Truncated    bool
	EffectsKnown bool
}

// Runner executes a task's command inside the workspace.
type Runner interface {
	Run(ctx context.Context, task domain.Task) (RunResult, error)
}
