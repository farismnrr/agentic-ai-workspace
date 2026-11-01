package application

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const (
	zipSessionFolder = ".workspace/resumable-uploads"
	zipIdleTTL       = 24 * time.Hour
	zipLifetime      = 72 * time.Hour
)

var (
	ErrZIPExpired  = errors.New("upload session expired")
	ErrZIPQuota    = errors.New("project upload storage quota exceeded")
	ErrZIPConflict = errors.New("upload offset mismatch")
	ErrZIPState    = errors.New("upload is no longer writable")
)

// ZIPSession is persisted inside the project's confined workspace. The opaque
// identifier never grants access: every HTTP operation rechecks auth + owner.
type ZIPSession struct {
	ID      string    `json:"id"`
	Owner   string    `json:"owner"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	Offset  int64     `json:"offset"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	State   string    `json:"state"`
	Path    string    `json:"path,omitempty"`
	SHA256  string    `json:"sha256,omitempty"`
}

type UploadChunkWriter interface {
	AppendUpload(rel string, offset, maxBytes int64, src io.Reader) (int64, error)
	TruncateUpload(rel string, size int64) error
}

func sessionMeta(id string) string { return path.Join(zipSessionFolder, id+".json") }
func sessionPart(id string) string { return path.Join(zipSessionFolder, id+".part") }

func validZIPID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, ch := range id {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func (e *Editor) readZIPSession(id, owner string) (ZIPSession, error) {
	if !validZIPID(id) {
		return ZIPSession{}, domain.ErrNotFound
	}
	bytes, truncated, err := e.files.ReadFile(sessionMeta(id), 4096)
	if err != nil || truncated {
		return ZIPSession{}, domain.ErrNotFound
	}
	var session ZIPSession
	if json.Unmarshal(bytes, &session) != nil || session.ID != id || session.Owner != owner {
		return ZIPSession{}, domain.ErrNotFound
	}
	if session.State != "ready" &&
		(time.Since(session.Updated) > zipIdleTTL || time.Since(session.Created) > zipLifetime) {
		return ZIPSession{}, ErrZIPExpired
	}
	if session.State == "uploading" {
		if info, statErr := e.files.Stat(sessionPart(id)); statErr == nil {
			if info.Size < 0 || info.Size > session.Size {
				return ZIPSession{}, ErrZIPState
			}
			session.Offset = info.Size // disk is the authoritative committed offset
		} else if !errors.Is(statErr, domain.ErrNotFound) {
			return ZIPSession{}, statErr
		} else {
			session.Offset = 0
		}
	}
	return session, nil
}

func (e *Editor) saveZIPSession(session ZIPSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return e.writer.WriteFile(sessionMeta(session.ID), data)
}

// StartZIP reserves the full declared compressed size, not only chunks already
// transferred, so concurrent sessions cannot exceed the project quota.
func (e *Editor) StartZIP(owner, name string, size int64) (ZIPSession, error) {
	if size < 22 {
		return ZIPSession{}, domain.ErrEmpty
	}
	if size > domain.MaxZIPBytes {
		return ZIPSession{}, domain.ErrTooLarge
	}
	if !strings.EqualFold(path.Ext(name), ".zip") {
		return ZIPSession{}, domain.ErrUnsupported
	}
	if err := e.reserveZIP(size); err != nil {
		return ZIPSession{}, err
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return ZIPSession{}, err
	}
	id := hex.EncodeToString(raw)
	now := time.Now().UTC()
	session := ZIPSession{ID: id, Owner: owner, Name: domain.SafeBase(name, "archive") + ".zip",
		Size: size, Created: now, Updated: now, State: "uploading"}
	if err := e.saveZIPSession(session); err != nil {
		return ZIPSession{}, err
	}
	return session, nil
}

func (e *Editor) reserveZIP(size int64) error {
	// Keep at least 128 MiB free after reserving the full compressed ZIP.
	// Never attempt a large stream merely because the per-project quota permits it.
	if capacity, ok := e.writer.(interface{ AvailableUploadBytes() (int64, error) }); ok {
		free, err := capacity.AvailableUploadBytes()
		if err != nil {
			return err
		}
		if free < size+(128<<20) {
			return ErrZIPQuota
		}
	}
	var used int64
	existing, err := e.files.ReadDir("uploads")
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	for _, file := range existing {
		if !file.Dir {
			used += file.Size
		}
	}
	sessions, err := e.files.ReadDir(zipSessionFolder)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	now := time.Now()
	active := 0
	for _, file := range sessions {
		if !strings.HasSuffix(file.Path, ".json") || file.Dir {
			continue
		}
		data, truncated, readErr := e.files.ReadFile(file.Path, 4096)
		if readErr != nil || truncated {
			return ErrZIPQuota
		}
		var session ZIPSession
		if json.Unmarshal(data, &session) != nil || !validZIPID(session.ID) {
			return ErrZIPQuota
		}
		expired := now.Sub(session.Updated) > zipIdleTTL || now.Sub(session.Created) > zipLifetime
		if expired {
			if err := e.writer.Remove(file.Path); err != nil {
				return err
			}
			if err := e.writer.Remove(sessionPart(session.ID)); err != nil && !errors.Is(err, domain.ErrNotFound) {
				return err
			}
			continue
		}
		if session.State == "uploading" || session.State == "validating" || session.State == "publishing" {
			active++
			used += session.Size
		}
	}
	if active >= 1 || used > domain.MaxProjectUploads || size > domain.MaxProjectUploads-used {
		return ErrZIPQuota
	}
	return nil
}

func (e *Editor) ZIPStatus(id, owner string) (ZIPSession, error) {
	return e.readZIPSession(id, owner)
}

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
