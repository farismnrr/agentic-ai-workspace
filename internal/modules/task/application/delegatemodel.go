package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// pickModel is the role's own model when it has one the person can reach, else the person's default,
// else the model the task started with. No model is named here. A plan is reached when the person is signed in
// to it; any other provider, when its key still works.
func (r *Runner) pickModel(ctx context.Context, task domain.Task, role AgentRole) (string, string, domain.Effort) {
	if role.ProviderID != "" && role.Model != "" && r.reachable(ctx, task.UserID, role.ProviderID) {
		return role.ProviderID, role.Model, role.Effort
	}

	providerID, model, err := r.deps.Models.Default(ctx, task.UserID)
	if err == nil && providerID != "" && model != "" && r.reachable(ctx, task.UserID, providerID) {
		return providerID, model, r.deps.Models.DefaultEffort(ctx, task.UserID)
	}

	return task.ProviderID, task.Model, task.Effort
}

// reachable says whether a teammate can run on this provider right now.
func (r *Runner) reachable(ctx context.Context, userID, providerID string) bool {
	if program, yes := domain.PlanRuntime(providerID); yes {
		if r.deps.CLI == nil {
			return false
		}

		signedIn, err := r.deps.CLI.Ready(ctx, userID, program)

		return err == nil && signedIn
	}

	_, err := r.deps.Models.Client(ctx, userID, providerID)

	return err == nil
}

func onPlan(providerID string) bool {
	_, yes := domain.PlanRuntime(providerID)

	return yes
}
