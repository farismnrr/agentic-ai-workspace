// Package notefiles exports task documents into the project's confined filesystem.
package notefiles

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"sync"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

var taskIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

const (
	directoryMode = 0o750
	documentMode  = 0o600
)

var errInvalidTaskID = errors.New("invalid task id")

// Store writes documents atomically so readers never observe a partially written file.
type Store struct {
	dir func(string) (string, error)
	mu  sync.Mutex
}

// New uses the same project directory resolver as the coding tools.
func New(dir func(string) (string, error)) *Store { return &Store{dir: dir} }

// Write confines every path and symlink to its project, including temporary files.
func (s *Store) Write(ctx context.Context, projectID string, files []application.NoteFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := s.dir(projectID)
	if err != nil {
		return fmt.Errorf("resolve notes folder: %w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open notes folder: %w", err)
	}
	defer func() { _ = root.Close() }()
	for _, file := range files {
		if err = ctx.Err(); err != nil {
			return fmt.Errorf("write notes: %w", err)
		}
		if file.Content == "" {
			continue
		} // Input documents already exist; never replace their original contents.
		if err = write(root, file); err != nil {
			return err
		}
	}
	return nil
}

// RemoveTask removes only one validated task's exported documents inside its project root.
func (s *Store) RemoveTask(ctx context.Context, projectID, taskID string) error {
	if !taskIDPattern.MatchString(taskID) {
		return errInvalidTaskID
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("remove task documents: %w", err)
	}
	dir, err := s.dir(projectID)
	if err != nil {
		return fmt.Errorf("resolve notes folder: %w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open notes folder: %w", err)
	}
	defer func() { _ = root.Close() }()
	if err := root.RemoveAll("project-notes/tasks/" + taskID); err != nil {
		return fmt.Errorf("remove task documents: %w", err)
	}
	return nil
}

func write(root *os.Root, file application.NoteFile) error {
	if err := root.MkdirAll(path.Dir(file.Path), directoryMode); err != nil {
		return fmt.Errorf("create notes folder: %w", err)
	}
	temp := file.Path + "." + rand.Text() + ".tmp"
	out, err := root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, documentMode)
	if err != nil {
		return fmt.Errorf("create document: %w", err)
	}
	defer func() { _ = root.Remove(temp) }()
	_, err = out.WriteString(file.Content)
	closed := out.Close()
	if err != nil {
		return fmt.Errorf("write document: %w", err)
	}
	if closed != nil {
		return fmt.Errorf("close document: %w", closed)
	}
	if err := root.Rename(temp, file.Path); err != nil {
		return fmt.Errorf("replace document: %w", err)
	}
	return nil
}
