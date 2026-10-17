package application

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Applied reports a finished modification and how to reverse it.
type Applied struct {
	Detail string
	Change domain.Change
}

// Editor makes reversible changes to workspace files. Changes are serialised
// so two tool calls cannot interleave their read-modify-write steps.
type Editor struct {
	files   FileSystem
	writer  Writer
	backups Backups
	ids     IDs
	mu      sync.Mutex
}

// NewEditor wires the write-side use cases.
func NewEditor(files FileSystem, writer Writer, backups Backups, ids IDs) *Editor {
	return &Editor{files: files, writer: writer, backups: backups, ids: ids}
}

// Edit replaces old with new in a text file; old must be unique unless all is set.
func (e *Editor) Edit(raw, oldText, newText string, all bool) (Applied, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rel, current, err := e.load(raw)
	if err != nil {
		return Applied{}, err
	}

	updated, count, err := replace(current, oldText, newText, all)
	if err != nil {
		return Applied{}, err
	}

	change, err := e.apply(rel, domain.OpEdit, current, []byte(updated), true)
	if err != nil {
		return Applied{}, err
	}

	return Applied{Detail: fmt.Sprintf("replaced %d occurrence(s)", count), Change: change}, nil
}

func replace(content, oldText, newText string, all bool) (string, int, error) {
	count := strings.Count(content, oldText)

	switch {
	case oldText == "" || count == 0:
		return "", 0, domain.ErrNoMatch
	case count > 1 && !all:
		return "", 0, domain.ErrNotUnique
	}

	if !all {
		return strings.Replace(content, oldText, newText, 1), 1, nil
	}

	return strings.ReplaceAll(content, oldText, newText), count, nil
}

// Write creates a file or replaces its whole content.
func (e *Editor) Write(raw, content string) (Applied, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rel, err := writablePath(raw)
	if err != nil {
		return Applied{}, err
	}

	if len(content) > domain.MaxWriteBytes {
		return Applied{}, domain.ErrTooLarge
	}

	previous, existed, err := e.previous(rel)
	if err != nil {
		return Applied{}, err
	}

	change, err := e.apply(rel, domain.OpWrite, previous, []byte(content), existed)
	if err != nil {
		return Applied{}, err
	}

	return Applied{Detail: fmt.Sprintf("wrote %d bytes", len(content)), Change: change}, nil
}

// Delete removes one file, keeping a backup.
func (e *Editor) Delete(raw string) (Applied, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rel, current, err := e.load(raw)
	if err != nil {
		return Applied{}, err
	}

	backup, err := e.backup(current)
	if err != nil {
		return Applied{}, err
	}

	if err := e.writer.Remove(rel); err != nil {
		return Applied{}, fmt.Errorf("delete %s: %w", rel, err)
	}

	change := domain.Change{Op: domain.OpDelete, Path: rel, Existed: true, BackupID: backup}

	return Applied{Detail: "deleted", Change: change}, nil
}

// DeleteFolder removes a folder and everything in it. It is for the person's own file browser: unlike a single
// file, a folder is not backed up, so the page asks first and says it cannot be undone. The tools never call it.
func (e *Editor) DeleteFolder(raw string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rel, err := writablePath(raw)
	if err != nil {
		return err
	}

	entry, err := e.files.Stat(rel)
	if err != nil {
		return fmt.Errorf("stat %s: %w", rel, err)
	}

	if !entry.Dir {
		return domain.ErrNotDir
	}

	if err = e.writer.RemoveAll(rel); err != nil {
		return fmt.Errorf("delete folder %s: %w", rel, err)
	}

	return nil
}

// Move renames a file without ever overwriting.
func (e *Editor) Move(rawFrom, rawTo string) (Applied, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	from, err := writablePath(rawFrom)
	if err != nil {
		return Applied{}, err
	}

	to, err := writablePath(rawTo)
	if err != nil {
		return Applied{}, err
	}

	if err := e.requireFile(from); err != nil {
		return Applied{}, err
	}

	if _, err := e.files.Stat(to); err == nil {
		return Applied{}, domain.ErrExists
	}

	if err := e.writer.Rename(from, to); err != nil {
		return Applied{}, fmt.Errorf("move %s: %w", from, err)
	}

	return Applied{Detail: "moved to " + to, Change: domain.Change{Op: domain.OpMove, Path: from, To: to}}, nil
}

func writablePath(raw string) (string, error) {
	rel, err := domain.CleanPath(raw)
	if err != nil {
		return "", fmt.Errorf("path: %w", err)
	}

	if rel == "." {
		return "", domain.ErrIsDir
	}

	return rel, nil
}

func hashOf(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}
