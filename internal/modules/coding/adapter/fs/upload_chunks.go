package fs

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// AppendUpload writes a single bounded chunk to a confined staging file.
// The caller serializes operations per project and checks the expected offset.
// Sync makes a successful upload-offset response durable across process restarts.
func (r *Root) AppendUpload(rel string, offset, maxBytes int64, src io.Reader) (int64, error) {
	if err := r.root.MkdirAll(path.Dir(rel), dirMode); err != nil {
		return 0, translate(err)
	}
	file, err := r.root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return 0, translate(err)
	}
	defer func() { _ = file.Close() }()
	stat, err := file.Stat()
	if err != nil {
		return 0, translate(err)
	}
	if !stat.Mode().IsRegular() {
		return 0, domain.ErrDenied
	}
	if stat.Size() != offset {
		return 0, domain.ErrStale
	}
	if offset < 0 || maxBytes <= 0 || maxBytes > domain.MaxZIPChunkBytes {
		return 0, domain.ErrTooLarge
	}
	n, err := io.Copy(file, io.LimitReader(src, maxBytes+1))
	if err != nil {
		return n, fmt.Errorf("append upload: %w", err)
	}
	if n > maxBytes {
		return n, domain.ErrTooLarge
	}
	if err := file.Sync(); err != nil {
		return n, fmt.Errorf("sync upload: %w", err)
	}
	return n, nil
}

// TruncateUpload rolls back an incomplete or rejected chunk, never advancing
// the committed offset after a failed request.
func (r *Root) TruncateUpload(rel string, size int64) error {
	file, err := r.root.OpenFile(rel, os.O_WRONLY, 0)
	if err != nil {
		return translate(err)
	}
	defer func() { _ = file.Close() }()
	if err := file.Truncate(size); err != nil {
		return translate(err)
	}
	return translate(file.Sync())
}
