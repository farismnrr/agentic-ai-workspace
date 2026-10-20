package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

// Plans are the vendors' own coding programs a person signed in to with their own plan. They are listed
// among the providers, with the models the program takes, so every screen treats them like any other provider.
type Plans interface {
	// Connected lists the plans the person is signed in to, each with its switch and its models.
	Connected(ctx context.Context, userID string) []domain.Provider
	SetEnabled(ctx context.Context, userID, id string, enabled bool) error
	SignOut(ctx context.Context, userID, id string) error
}

// WithPlans lists the person's own plans among their providers.
func (s *Service) WithPlans(plans Plans) *Service {
	s.plans = plans

	return s
}

// plan finds the person's signed-in plan with this id.
func (s *Service) plan(ctx context.Context, userID, id string) (domain.Provider, error) {
	if s.plans != nil {
		for _, plan := range s.plans.Connected(ctx, userID) {
			if plan.ID == id {
				return plan, nil
			}
		}
	}

	return domain.Provider{}, fmt.Errorf("find plan: %w", domain.ErrNotFound)
}
