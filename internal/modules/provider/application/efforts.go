package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

// EffortLevels are the effort levels a person's model takes; none when the model does not reason or is unknown.
func (s *Service) EffortLevels(ctx context.Context, userID, providerID, model string) []string {
	providers, err := s.List(ctx, userID)
	if err != nil {
		return nil
	}

	for _, provider := range providers {
		if provider.ID != providerID {
			continue
		}

		info, ok := provider.Catalog[model]
		if !ok {
			info = domain.ModelInfo{Name: model}
		}

		return domain.EffortLevels(provider.Kind, info)
	}

	return nil
}
