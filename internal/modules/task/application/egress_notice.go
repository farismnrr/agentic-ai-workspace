package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// NoteEgress appends a person-visible notice to a task's existing event stream.
// Only trusted server components may call this; the HTTP adapter has already verified grant scope.
func (s *Service) NoteEgress(ctx context.Context, taskID, text string) error {
	_, err := s.deps.Events.AppendEvent(ctx, taskID, domain.Event{Kind: domain.EventMessage, Text: text})
	if err != nil {
		return fmt.Errorf("append egress notice: %w", err)
	}
	return nil
}
