package application

import (
	"fmt"
	"io"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Raw is a whole file as bytes, for a download or a picture the agent looks at.
type Raw struct {
	Path string
	Data []byte
}

// RawStream is a confined file handle suitable for downloads and media playback.
type RawStream struct {
	File io.ReadSeekCloser
	Path string
	Size int64
}

// OpenRaw returns a file-backed stream; the caller must close its handle.
func (s *Service) OpenRaw(raw string) (RawStream, error) {
	rel, err := domain.CleanPath(raw)
	if err != nil {
		return RawStream{}, fmt.Errorf("read: %w", err)
	}

	file, size, err := s.files.Open(rel)
	if err != nil {
		return RawStream{}, fmt.Errorf("read %s: %w", rel, err)
	}
	return RawStream{Path: rel, File: file, Size: size}, nil
}

// ReadRaw returns a whole file of any kind, or domain.ErrTooLarge when it is bigger than limit bytes.
func (s *Service) ReadRaw(raw string, limit int64) (Raw, error) {
	rel, err := domain.CleanPath(raw)
	if err != nil {
		return Raw{}, fmt.Errorf("read: %w", err)
	}

	data, truncated, err := s.files.ReadFile(rel, limit)
	if err != nil {
		return Raw{}, fmt.Errorf("read %s: %w", rel, err)
	}

	if truncated {
		return Raw{}, domain.ErrTooLarge
	}

	return Raw{Path: rel, Data: data}, nil
}
