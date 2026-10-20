// Package application holds the use cases of extensions.
package application

import (
	"context"
	"io"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/domain"
)

// Store keeps one row per project's extension.
type Store interface {
	Save(ctx context.Context, extension domain.Extension) error
	ByProject(ctx context.Context, userID, projectID string) (domain.Extension, error)
	List(ctx context.Context, userID string) ([]domain.Extension, error)
	Delete(ctx context.Context, userID, projectID string) error
	// DeleteProject removes a project's extension whoever owns it, and returns the owner (empty when there was none).
	DeleteProject(ctx context.Context, projectID string) (userID string, err error)
}

// Tree is what is in the folder an extension is built into.
type Tree struct {
	// Has says whether the folder holds a file at this path (relative to the folder).
	Has      func(path string) bool
	Manifest []byte
	Size     int64
	Files    int
}

// Packer reads a project's build folder and keeps the package the person downloads.
type Packer interface {
	// Inspect reads the folder, failing with domain.ErrBadFolder, ErrNoManifest or ErrTooLarge.
	Inspect(ctx context.Context, projectID, folder string) (Tree, error)
	// Pack zips the folder into the project's package, replacing the last one.
	Pack(ctx context.Context, projectID, folder string) error
	// Open returns the project's package and its size.
	Open(projectID string) (io.ReadCloser, int64, error)
	// Remove deletes the project's package; a missing one is not an error.
	Remove(projectID string) error
}

// Projects says who owns a project.
type Projects interface {
	// Owner returns the owner and project ID of a project that exists and is not archived.
	Owner(ctx context.Context, projectID string) (userID, slug string, err error)
}

// Notifier tells the person's open pages that their extensions changed.
type Notifier interface {
	Changed(userID string)
}
