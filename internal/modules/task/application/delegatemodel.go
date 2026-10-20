package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// pickModel is the role's own model when it has one the person can reach, else the person's default,
// else the model the task started with. No model is named here. A teammate runs on the app's own loop, which
// a plan (the vendor's own program) cannot serve, so a plan is never picked for one.
func (r *Runner) pickModel(ctx context.Context, task domain.Task, role AgentRole) (string, string, domain.Effort) {
	if role.ProviderID != "" && role.Model != "" && !onPlan(role.ProviderID) {
		if _, err := r.deps.Models.Client(ctx, task.UserID, role.ProviderID); err == nil {
			return role.ProviderID, role.Model, role.Effort
		}
	}

	providerID, model, err := r.deps.Models.Default(ctx, task.UserID)
	if err == nil && providerID != "" && model != "" && !onPlan(providerID) {
		return providerID, model, r.deps.Models.DefaultEffort(ctx, task.UserID)
	}

	return task.ProviderID, task.Model, task.Effort
}

func onPlan(providerID string) bool {
	_, yes := domain.PlanRuntime(providerID)

	return yes
}
