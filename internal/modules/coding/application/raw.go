package application

import (
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Raw is a whole file as bytes, for a download or a picture the agent looks at.
type Raw struct {
	Path string
	Data []byte
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
