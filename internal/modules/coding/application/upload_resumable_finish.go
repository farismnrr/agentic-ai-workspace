package application

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// AppendZIP uses a confined append-only file, fsyncs before acknowledging the
// new offset, and truncates rejected chunks. The HTTP adapter serializes calls.
func (e *Editor) AppendZIP(id, owner string, offset int64, chunk io.Reader, length int64) (ZIPSession, error) {
	session, err := e.readZIPSession(id, owner)
	if err != nil {
		return ZIPSession{}, err
	}
	if session.State != "uploading" {
		return session, ErrZIPState
	}
	if offset != session.Offset {
		return session, ErrZIPConflict
	}
	if length <= 0 || length > domain.MaxZIPChunkBytes || length > session.Size-offset {
		return session, domain.ErrTooLarge
	}
	writer, ok := e.writer.(UploadChunkWriter)
	if !ok {
		return session, fmt.Errorf("upload storage does not support durable append")
	}
	n, err := writer.AppendUpload(sessionPart(id), offset, length, chunk)
	if err != nil || n != length {
		if rollErr := writer.TruncateUpload(sessionPart(id), offset); rollErr != nil {
			return session, fmt.Errorf("upload rollback failed: %w", rollErr)
		}
		if err != nil {
			return session, err
		}
		return session, domain.ErrEmpty
	}
	session.Offset += n
	if session.Offset == session.Size {
		session.State = "validating"
	}
	session.Updated = time.Now().UTC()
	if err := e.saveZIPSession(session); err != nil {
		return session, err
	}
	return session, nil
}

func (e *Editor) CancelZIP(id, owner string) error {
	session, err := e.readZIPSession(id, owner)
	if err != nil {
		return err
	}
	if session.State == "publishing" || session.State == "ready" {
		return ErrZIPState
	}
	if err := e.writer.Remove(sessionMeta(id)); err != nil {
		return err
	}
	if err := e.writer.Remove(sessionPart(id)); err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	return nil
}

// FinishZIP is idempotent; do not expose a path until validation and rename
// succeeded. A restart after rename but before state persistence can recover.
func (e *Editor) FinishZIP(id, owner string) (ZIPSession, error) {
	session, err := e.readZIPSession(id, owner)
	if err != nil {
		return ZIPSession{}, err
	}
	if session.State == "ready" {
		return session, nil
	}
	if session.State != "uploading" && session.State != "validating" && session.State != "publishing" {
		return session, ErrZIPState
	}
	if session.State == "uploading" || session.State == "validating" {
		if session.Offset != session.Size {
			return session, ErrZIPState
		}
		if err := e.validateArchive(sessionPart(id), session.Size); err != nil {
			_ = e.CancelZIP(id, owner)
			return session, err
		}
		file, _, err := e.files.Open(sessionPart(id))
		if err != nil {
			return session, err
		}
		digest := sha256.New()
		_, err = io.Copy(digest, file)
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			return session, fmt.Errorf("hash staged ZIP: %v %v", err, closeErr)
		}
		session.SHA256 = hex.EncodeToString(digest.Sum(nil))
		session.Path = "uploads/" + strings.TrimSuffix(session.Name, ".zip") + "-" + id + ".zip"
		session.State = "publishing"
		session.Updated = time.Now().UTC()
		if err := e.saveZIPSession(session); err != nil {
			return session, err
		}
	}
	// Recovery-safe: if the process died after rename, the destination exists
	// and is already verified. Never copy a complete archive a second time.
	if info, err := e.files.Stat(session.Path); err == nil {
		if info.Size != session.Size {
			return session, ErrZIPState
		}
	} else if errors.Is(err, domain.ErrNotFound) {
		if err := e.writer.Rename(sessionPart(id), session.Path); err != nil {
			return session, err
		}
	} else {
		return session, err
	}
	session.State = "ready"
	session.Offset = session.Size
	session.Updated = time.Now().UTC()
	if err := e.saveZIPSession(session); err != nil {
		return session, err
	}
	return session, nil
}
