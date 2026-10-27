package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

// Legacy 9router connections cannot be used, but the user's sealed keys remain
// available for explicit deletion. This avoids destructive migrations.
func disableLegacyAccountRelays(providers []domain.Provider) {
	for i := range providers {
		if domain.IsLegacy9router(providers[i].Kind, providers[i].Name, providers[i].BaseURL) {
			providers[i].Enabled = false
		}
	}
}

func (s *Service) checkLegacyAccountRelay(ctx context.Context, userID, id string) error {
	provider, _, err := s.store.Find(ctx, userID, id)
	if err != nil {
		return fmt.Errorf("find legacy provider: %w", err)
	}
	if domain.IsLegacy9router(provider.Kind, provider.Name, provider.BaseURL) {
		return domain.ErrAccountRelay
	}
	return nil
}
