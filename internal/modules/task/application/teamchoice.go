package application

import "context"

// chooseTeam gives every new task a team, as long as the app has teams at all. Only code that starts a task on
// someone's behalf can ask for none: the person has no switch.
func (s *Service) chooseTeam(asked *bool) bool {
	if s.deps.Team == nil || s.deps.Scopes == nil || s.deps.Delegations == nil {
		return false
	}

	if asked != nil {
		return *asked
	}

	return true
}

// leadModel fills in the orchestrator's own model when a team task did not pick one, so the lead role's
// model is used before the person's default.
func (s *Service) leadModel(ctx context.Context, userID string, in CreateInput, team bool) CreateInput {
	if !team || (in.ProviderID != "" && in.Model != "") {
		return in
	}

	roster, err := s.deps.Team.Roster(ctx, userID)
	if err != nil {
		return in
	}

	lead := s.leadTier(ctx, userID, roster.Lead)
	in.ProviderID, in.Model = lead.ProviderID, lead.Model
	if in.Effort == "" {
		in.Effort = lead.Effort
	}

	return in
}

func (s *Service) leadTier(ctx context.Context, userID string, lead AgentRole) AgentRole {
	if lead.ProviderID != "" && lead.Model != "" {
		return lead
	}
	lead.ProviderID, lead.Model = "", ""
	if lead.Tier != TierThinking && lead.Tier != TierFast {
		return lead
	}
	provider, model, effort, ok := s.deps.Models.Tier(ctx, userID, lead.Tier)
	if ok {
		lead.ProviderID, lead.Model, lead.Effort = provider, model, effort
	}
	return lead
}

// followsSettings distinguishes an ordinary composer request from an explicit API selection.
func (in CreateInput) followsSettings() bool {
	return in.ProviderID == "" && in.Model == "" && in.Effort == ""
}
