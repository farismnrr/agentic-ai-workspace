package application

import (
	"context"
	"fmt"
	"slices"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

// SetContextWindow records how much one of a provider's models can hold at once.
func (s *Service) SetContextWindow(ctx context.Context, userID, id, model string, tokens int64) error {
	if !domain.ValidContextWindow(tokens) {
		return domain.ErrInvalidContext
	}

	provider, _, err := s.store.Find(ctx, userID, id)
	if err != nil {
		return fmt.Errorf("find provider: %w", err)
	}

	if !slices.Contains(provider.Models, model) {
		return domain.ErrNotFound
	}

	if err := s.store.SetContextWindow(ctx, userID, id, model, tokens); err != nil {
		return fmt.Errorf("save context window: %w", err)
	}

	return nil
}

// ContextWindow is how much the given model can hold at once, in tokens.
func (s *Service) ContextWindow(ctx context.Context, userID, id, model string) int64 {
	provider, _, err := s.store.Find(ctx, userID, id)
	if err != nil {
		return domain.ContextWindow(model, nil, nil)
	}

	return domain.ContextWindow(model, provider.ContextWindows, provider.ReportedWindows)
}
