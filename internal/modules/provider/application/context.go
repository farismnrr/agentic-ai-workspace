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

	if domain.IsPlanID(id) {
		return s.setPlanWindow(ctx, userID, id, model, tokens)
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
	if domain.IsPlanID(id) {
		if plan, err := s.plan(ctx, userID, id); err == nil {
			return domain.ContextWindow(model, plan.ContextWindows, plan.Catalog)
		}
	}

	provider, _, err := s.store.Find(ctx, userID, id)
	if err != nil {
		return domain.ContextWindow(model, nil, nil)
	}

	return domain.ContextWindow(model, provider.ContextWindows, provider.Catalog)
}

func (s *Service) setPlanWindow(ctx context.Context, userID, id, model string, tokens int64) error {
	plan, err := s.plan(ctx, userID, id)
	if err != nil {
		return err
	}

	if !slices.Contains(plan.Models, model) {
		return domain.ErrNotFound
	}

	if err = s.plans.SetWindow(ctx, userID, id, model, tokens); err != nil {
		return fmt.Errorf("save context window: %w", err)
	}

	return nil
}
