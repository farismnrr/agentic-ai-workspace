package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// pickModel is the role's own model when it has one the person can reach, else its tier's, else the default,
// else the model the task started with. No model is named here. A plan is reached when the person is signed in
// to it; any other provider, when its key still works.
func (r *Runner) pickModel(ctx context.Context, task domain.Task, role AgentRole) (string, string, domain.Effort) {
	if role.ProviderID != "" && role.Model != "" && r.reachable(ctx, task.UserID, role.ProviderID) {
		return role.ProviderID, role.Model, role.Effort
	}

	if providerID, model, effort, ok := r.tierModel(ctx, task.UserID, role.Tier); ok {
		return providerID, model, effort
	}

	providerID, model, err := r.deps.Models.Default(ctx, task.UserID)
	if err == nil && providerID != "" && model != "" && r.reachable(ctx, task.UserID, providerID) {
		return providerID, model, r.deps.Models.DefaultEffort(ctx, task.UserID)
	}

	return task.ProviderID, task.Model, task.Effort
}

// tierModel is the model the person set for the Thinking or the Fast tier, when they set one they can reach.
func (r *Runner) tierModel(ctx context.Context, userID, tier string) (string, string, domain.Effort, bool) {
	if tier != TierThinking && tier != TierFast {
		return "", "", domain.EffortDefault, false
	}

	providerID, model, effort, ok := r.deps.Models.Tier(ctx, userID, tier)
	if !ok || !r.reachable(ctx, userID, providerID) {
		return "", "", domain.EffortDefault, false
	}

	return providerID, model, effort, true
}

// shownWith is the job as the card shows it: with the effort the orchestrator asked for when the teammate's own
// is left to the model, so the card never says "auto".
func shownWith(d Delegation, asked string) Delegation {
	if d.Effort == domain.EffortDefault {
		d.Effort = domain.Effort(asked)
	}

	return d
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

// fitEffort is the effort a teammate works at: the one the person or the role set, else the one the lead asked
// for, fitted to the levels the teammate's model takes. Left empty, the model uses its own.
func (r *Runner) fitEffort(
	ctx context.Context, userID, providerID, model string, set domain.Effort, asked string,
) domain.Effort {
	if set != domain.EffortDefault || asked == "" || r.deps.Efforts == nil {
		return set
	}

	return domain.Nearest(r.deps.Efforts.EffortLevels(ctx, userID, providerID, model), domain.Effort(asked))
}

func onPlan(providerID string) bool {
	_, yes := domain.PlanRuntime(providerID)

	return yes
}
