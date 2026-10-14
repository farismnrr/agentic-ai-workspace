package application

import (
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Undo reverses a recorded change, unless the files were touched again since.
func (e *Editor) Undo(change domain.Change) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch change.Op {
	case domain.OpEdit, domain.OpWrite:
		return e.undoContent(change)
	case domain.OpDelete:
		return e.undoDelete(change)
	case domain.OpMove:
		return e.undoMove(change)
	}

	return domain.ErrNotUndoable
}

func (e *Editor) undoContent(change domain.Change) error {
	data, _, err := e.files.ReadFile(change.Path, domain.MaxWriteBytes)
	if err != nil || hashOf(data) != change.AfterHash {
		return domain.ErrChanged
	}

	if !change.Existed {
		return e.remove(change.Path)
	}

	return e.restore(change)
}

func (e *Editor) undoDelete(change domain.Change) error {
	if _, err := e.files.Stat(change.Path); !errors.Is(err, domain.ErrNotFound) {
		return domain.ErrChanged
	}

	return e.restore(change)
}

func (e *Editor) undoMove(change domain.Change) error {
	if _, err := e.files.Stat(change.To); err != nil {
		return domain.ErrChanged
	}

	if _, err := e.files.Stat(change.Path); !errors.Is(err, domain.ErrNotFound) {
		return domain.ErrChanged
	}

	if err := e.writer.Rename(change.To, change.Path); err != nil {
		return fmt.Errorf("move back: %w", err)
	}

	return nil
}

func (e *Editor) restore(change domain.Change) error {
	old, err := e.backups.Load(change.BackupID)
	if err != nil {
		return fmt.Errorf("load backup: %w", err)
	}

	if err := e.writer.WriteFile(change.Path, old); err != nil {
		return fmt.Errorf("restore %s: %w", change.Path, err)
	}

	return nil
}

func (e *Editor) remove(rel string) error {
	if err := e.writer.Remove(rel); err != nil {
		return fmt.Errorf("remove %s: %w", rel, err)
	}

	return nil
}
