package application

import (
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Delete removes text files with undo support; binary and oversized files have no backup.
func (e *Editor) Delete(raw string) (Applied, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rel, err := writablePath(raw)
	if err != nil {
		return Applied{}, err
	}
	entry, err := e.files.Stat(rel)
	if err != nil {
		return Applied{}, fmt.Errorf("stat %s: %w", rel, err)
	}
	if entry.Dir {
		return Applied{}, domain.ErrIsDir
	}
	if entry.Size > domain.MaxWriteBytes {
		return e.deleteWithoutBackup(rel)
	}

	data, truncated, err := e.files.ReadFile(rel, domain.MaxWriteBytes)
	if err != nil {
		return Applied{}, fmt.Errorf("read %s: %w", rel, err)
	}
	if truncated || isBinary(data) {
		return e.deleteWithoutBackup(rel)
	}

	backup, err := e.backup(string(data))
	if err != nil {
		return Applied{}, err
	}
	if err := e.writer.Remove(rel); err != nil {
		return Applied{}, fmt.Errorf("delete %s: %w", rel, err)
	}

	change := domain.Change{Op: domain.OpDelete, Path: rel, Existed: true, BackupID: backup}
	return Applied{Detail: "deleted", Change: change}, nil
}

func (e *Editor) deleteWithoutBackup(rel string) (Applied, error) {
	if err := e.writer.Remove(rel); err != nil {
		return Applied{}, fmt.Errorf("delete %s: %w", rel, err)
	}
	return Applied{Detail: "deleted"}, nil
}

// DeleteFolder removes a folder for the file browser; the operation cannot be undone.
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
