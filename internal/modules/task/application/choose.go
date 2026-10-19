package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// chooseEffort uses the requested level, or the person's default when none was asked for.
func (s *Service) chooseEffort(ctx context.Context, userID string, asked domain.Effort) (domain.Effort, error) {
	if !asked.Valid() {
		return "", domain.ErrBadEffort
	}

	if asked != domain.EffortDefault {
		return asked, nil
	}

	if preferred := s.deps.Models.DefaultEffort(ctx, userID); preferred.Valid() {
		return preferred, nil
	}

	return domain.EffortDefault, nil
}

// chooseModel uses the requested model, or the person's default, and checks it can be reached.
func (s *Service) chooseModel(ctx context.Context, userID string, in CreateInput) (string, string, error) {
	providerID, model := in.ProviderID, in.Model

	if _, onPlan := domain.PlanRuntime(providerID); onPlan {
		return providerID, model, nil
	}

	if providerID == "" || model == "" {
		var err error
		if providerID, model, err = s.deps.Models.Default(ctx, userID); err != nil || providerID == "" || model == "" {
			return "", "", domain.ErrNoModel
		}
	}

	if _, err := s.deps.Models.Client(ctx, userID, providerID); err != nil {
		return "", "", domain.ErrNoModel
	}

	return providerID, model, nil
}

// planProgram says whether the task runs on the person's own plan with a vendor's coding program: they
// asked for it, or it is their default. Without a runner for programs, nothing does.
func (s *Service) planProgram(ctx context.Context, userID string, in CreateInput) (string, bool) {
	if s.deps.CLI == nil {
		return "", false
	}

	if in.ProviderID != "" {
		return domain.PlanRuntime(in.ProviderID)
	}

	providerID, _, err := s.deps.Models.Default(ctx, userID)
	if err != nil {
		return "", false
	}

	return domain.PlanRuntime(providerID)
}

// route settles whether a team does the task, and which provider it starts on: the person's own plan when
// that is what they asked for or prefer, the lead's model for a team, or the person's default model.
func (s *Service) route(ctx context.Context, userID string, in CreateInput) (CreateInput, bool) {
	program, onPlan := s.planProgram(ctx, userID, in)
	team := s.chooseTeam(in.Team) && !onPlan

	if onPlan {
		in.ProviderID, in.Model = domain.PlanProvider(program), domain.PlanModel
	}

	return s.leadModel(ctx, userID, in, team), team
}
