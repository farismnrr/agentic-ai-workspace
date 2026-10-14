package application

import (
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// load returns an existing, text, size-limited file for modification.
func (e *Editor) load(raw string) (string, string, error) {
	rel, err := writablePath(raw)
	if err != nil {
		return "", "", err
	}

	data, truncated, err := e.files.ReadFile(rel, domain.MaxWriteBytes)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", rel, err)
	}

	if truncated {
		return "", "", domain.ErrTooLarge
	}

	if isBinary(data) {
		return "", "", domain.ErrBinary
	}

	return rel, string(data), nil
}

// previous returns the current content of a path that may not exist yet.
func (e *Editor) previous(rel string) (string, bool, error) {
	entry, err := e.files.Stat(rel)
	if errors.Is(err, domain.ErrNotFound) {
		return "", false, nil
	}

	if err != nil {
		return "", false, fmt.Errorf("stat %s: %w", rel, err)
	}

	if entry.Dir {
		return "", false, domain.ErrIsDir
	}

	data, truncated, err := e.files.ReadFile(rel, domain.MaxWriteBytes)
	if err != nil || truncated {
		return "", false, errors.Join(domain.ErrTooLarge, err)
	}

	return string(data), true, nil
}

// apply backs up previous (when the file existed), writes next and describes the change.
func (e *Editor) apply(rel string, op domain.Op, previous string, next []byte, existed bool) (domain.Change, error) {
	change := domain.Change{Op: op, Path: rel, Existed: existed, AfterHash: hashOf(next)}

	if existed {
		id, err := e.backup(previous)
		if err != nil {
			return domain.Change{}, err
		}

		change.BackupID = id
	}

	if err := e.writer.WriteFile(rel, next); err != nil {
		return domain.Change{}, fmt.Errorf("write %s: %w", rel, err)
	}

	return change, nil
}

func (e *Editor) backup(content string) (string, error) {
	id, err := e.ids.ID()
	if err != nil {
		return "", fmt.Errorf("new backup id: %w", err)
	}

	if err := e.backups.Save(id, []byte(content)); err != nil {
		return "", fmt.Errorf("save backup: %w", err)
	}

	return id, nil
}

func (e *Editor) requireFile(rel string) error {
	entry, err := e.files.Stat(rel)
	if err != nil {
		return fmt.Errorf("stat %s: %w", rel, err)
	}

	if entry.Dir {
		return domain.ErrIsDir
	}

	return nil
}
