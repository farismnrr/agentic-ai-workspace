// Package http serves a person's own view of a project's files: the tree, a file, saving an edit and a stream that
// says when the files changed. The agent reaches the same files through its tools; this is for the browser.
package http

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
)

// Workspace is what the file screens use of one project.
type Workspace struct {
	Reader *application.Service
	Editor *application.Editor
	Git    *application.GitStatus
	Head   *application.GitHead
}

// Workspaces opens a project's files for the person who owns it; any other person gets domain.ErrNotFound.
type Workspaces interface {
	Open(ctx context.Context, userID, projectID string) (Workspace, error)
}

// Changes tells a viewer when a project's files may have changed.
type Changes interface {
	// Announce says the files changed, for the other pages looking at them.
	Announce(projectID string)
	// Follow returns a channel that receives a signal whenever they may have changed, and a function to stop.
	Follow(projectID string) (signals <-chan struct{}, stop func())
}
