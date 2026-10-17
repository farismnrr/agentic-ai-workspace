// Package shots keeps the screenshots of tasks as PNG files, one folder per task.
package shots

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	dirMode  = 0o750
	fileMode = 0o640
	idBytes  = 12
	// maxPNG keeps one screenshot well under what a model accepts (10 MB as base64).
	maxPNG = 6 << 20
)

var (
	errTooBig = errors.New("the screenshot is too large")
	namePart  = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	fileName  = regexp.MustCompile(`^[a-f0-9]{24}\.png$`)
)

// Store writes screenshots under one folder of the data directory.
type Store struct{ root string }

// New keeps screenshots in dir.
func New(dir string) *Store { return &Store{root: dir} }

// Save writes a PNG and returns the name to find it by.
func (s *Store) Save(_ context.Context, taskID string, png []byte) (string, error) {
	if !namePart.MatchString(taskID) || len(png) > maxPNG {
		return "", errTooBig
	}

	raw := make([]byte, idBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("name screenshot: %w", err)
	}

	id := hex.EncodeToString(raw) + ".png"
	dir := filepath.Join(s.root, taskID)

	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create screenshot folder: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, id), png, fileMode); err != nil {
		return "", fmt.Errorf("save screenshot: %w", err)
	}

	return id, nil
}

// Load reads a screenshot; a name that was never saved is domain.ErrNotFound. The folder is opened as a root, so
// a name can never lead out of it.
func (s *Store) Load(_ context.Context, taskID, id string) ([]byte, error) {
	if !namePart.MatchString(taskID) || !fileName.MatchString(id) {
		return nil, domain.ErrNotFound
	}

	root, err := os.OpenRoot(filepath.Join(s.root, taskID))
	if err != nil {
		return nil, domain.ErrNotFound
	}

	defer func() { _ = root.Close() }()

	file, err := root.Open(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxPNG+1))
	if err != nil {
		return nil, domain.ErrNotFound
	}

	return data, nil
}

// Remove deletes every screenshot of a task.
func (s *Store) Remove(_ context.Context, taskID string) error {
	if !namePart.MatchString(taskID) {
		return nil
	}

	if err := os.RemoveAll(filepath.Join(s.root, taskID)); err != nil {
		return fmt.Errorf("remove screenshots: %w", err)
	}

	return nil
}
