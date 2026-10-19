// Package application holds the use cases of sites.
package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// Store keeps one row per project's site.
type Store interface {
	// Save inserts or replaces the project's site.
	Save(ctx context.Context, site domain.Site) error
	// ByProject returns domain.ErrNotFound when the project never started a site.
	ByProject(ctx context.Context, projectID string) (domain.Site, error)
	// BySlug returns domain.ErrNotFound for an unknown address name.
	BySlug(ctx context.Context, slug string) (domain.Site, error)
	// ByToken returns domain.ErrNotFound for an unknown token (old addresses carried it).
	ByToken(ctx context.Context, token string) (domain.Site, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Site, error)
	// Running lists every site marked running.
	Running(ctx context.Context) ([]domain.Site, error)
	SetVisibility(ctx context.Context, projectID string, visibility domain.Visibility) error
	// MarkStopped marks one site stopped, or every site when projectID is empty.
	MarkStopped(ctx context.Context, projectID string) error
}

// StartSpec is everything the runtime needs to start an app.
type StartSpec struct {
	ProjectID string
	BasePath  string
	Command   string
	Port      int
}

// Runtime starts and stops the containers sites run in.
type Runtime interface {
	// Start runs the app and returns once it answers on its port, or fails with the reason.
	Start(ctx context.Context, spec StartSpec) (containerID string, err error)
	Stop(ctx context.Context, containerID string)
	Alive(ctx context.Context, containerID string) bool
	// Socket is the Unix socket the app is reachable through.
	Socket(projectID string) string
	// Get requests a path from the running app, as the proxy would, with the address prefix removed.
	Get(ctx context.Context, projectID, prefix, path string) (Page, error)
	// Capture runs the browser against the app and returns what it printed.
	Capture(ctx context.Context, req CaptureRequest) (output string, err error)
}

// Projects says who owns a project.
type Projects interface {
	// Owner returns the owner and name of a project that exists and is not archived.
	Owner(ctx context.Context, projectID string) (userID, name string, err error)
}

// Notifier tells the person's open pages that their sites changed.
type Notifier interface {
	Changed(userID string)
}

// IDs mints the random UUID that makes a site address unguessable.
type IDs interface {
	UUID() (string, error)
}

// Limits bound how many sites run at once.
type Limits struct {
	// PerUser caps the apps one person runs at once; zero means no cap, since a project has one app and the
	// person decides how many projects to run.
	PerUser int
	// Total caps the apps the server runs at once, to keep the machine alive; zero means no cap.
	Total int
}
