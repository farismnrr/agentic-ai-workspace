package files

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
)

func (m *Manager) Status(ctx context.Context, userID, projectID, id string) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	root, err := m.open(ctx, userID, projectID)
	if err != nil {
		return Session{}, err
	}
	defer root.Close()
	return m.read(root, userID, projectID, id)
}

func (m *Manager) Append(ctx context.Context, userID, projectID, id string, offset int64, encoded string) (Session, error) {
	if len(encoded) > base64.StdEncoding.EncodedLen(maxChunk) {
		return Session{}, errors.New("chunk too large")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) == 0 || len(data) > maxChunk {
		return Session{}, errors.New("invalid upload chunk")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	root, err := m.open(ctx, userID, projectID)
	if err != nil {
		return Session{}, err
	}
	defer root.Close()
	s, err := m.read(root, userID, projectID, id)
	if err != nil {
		return Session{}, err
	}
	if s.Status != "uploading" || offset != s.Offset || s.Offset+int64(len(data)) > s.Size {
		return Session{}, errors.New("upload offset or state mismatch")
	}
	file, err := root.OpenFile(partPath(id), os.O_WRONLY, 0)
	if err != nil {
		return Session{}, err
	}
	if _, err = file.Seek(offset, io.SeekStart); err != nil {
		_ = file.Close()
		return Session{}, err
	}
	n, err := file.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return Session{}, err
	}
	if closeErr != nil {
		return Session{}, closeErr
	}
	s.Offset += int64(len(data))
	return s, save(root, s)
}

func (m *Manager) Finish(ctx context.Context, userID, projectID, id string) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	root, err := m.open(ctx, userID, projectID)
	if err != nil {
		return Session{}, err
	}
	defer root.Close()
	s, err := m.read(root, userID, projectID, id)
	if err != nil {
		return Session{}, err
	}
	if s.Status == "done" {
		return s, nil
	}
	if s.Status != "uploading" || s.Offset != s.Size {
		return Session{}, errors.New("upload incomplete")
	}
	part, err := root.Open(partPath(id))
	if err != nil {
		return Session{}, err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, part)
	if err != nil {
		_ = part.Close()
		return Session{}, err
	}
	if hex.EncodeToString(hash.Sum(nil)) != s.SHA256 {
		_ = part.Close()
		return Session{}, errors.New("SHA-256 mismatch")
	}
	if err := validateBinary(part, s); err != nil {
		_ = part.Close()
		return Session{}, err
	}
	_ = part.Close()
	dest := path.Join("uploads", s.Name)
	// Root.Link atomically publishes a verified file with create-only semantics.
	// Both paths remain confined to this project's os.Root, including symlinks.
	if err := root.Link(partPath(id), dest); err != nil {
		// A retry after a crash between Link and session-save is idempotent
		// only if the destination is the exact same inode we validated.
		partInfo, pErr := root.Stat(partPath(id))
		destInfo, dErr := root.Stat(dest)
		if pErr != nil || dErr != nil || !os.SameFile(partInfo, destInfo) {
			return Session{}, fmt.Errorf("publish upload without overwrite: %w", err)
		}
	}
	s.Status, s.Path = "done", dest
	if err := save(root, s); err != nil {
		return Session{}, err
	}
	_ = root.Remove(partPath(id))
	if m.Notify != nil {
		m.Notify(projectID)
	}
	return s, nil
}

func (m *Manager) Cancel(ctx context.Context, userID, projectID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	root, err := m.open(ctx, userID, projectID)
	if err != nil {
		return err
	}
	defer root.Close()
	s, err := m.read(root, userID, projectID, id)
	if err != nil {
		return err
	}
	if s.Status == "done" {
		return errors.New("completed upload cannot be cancelled")
	}
	if err := root.Remove(partPath(id)); err != nil {
		return err
	}
	s.Status = "cancelled"
	return save(root, s)
}
