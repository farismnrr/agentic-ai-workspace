package application

import "context"

// chooseTeam uses the requested choice, or the person's default for new tasks. A task has a team only
// when the app has teams switched on at all.
func (s *Service) chooseTeam(ctx context.Context, userID string, asked *bool) bool {
	if s.deps.Team == nil || s.deps.Scopes == nil || s.deps.Delegations == nil {
		return false
	}

	if asked != nil {
		return *asked
	}

	return s.deps.Team.UseByDefault(ctx, userID)
}

// leadModel fills in the orchestrator's own model when a team task did not pick one, so the lead role's
// model is used before the person's default.
func (s *Service) leadModel(ctx context.Context, userID string, in CreateInput, team bool) CreateInput {
	if !team || (in.ProviderID != "" && in.Model != "") {
		return in
	}

	roster, err := s.deps.Team.Roster(ctx, userID)
	if err != nil || roster.Lead.ProviderID == "" || roster.Lead.Model == "" {
		return in
	}

	in.ProviderID, in.Model = roster.Lead.ProviderID, roster.Lead.Model
	if in.Effort == "" {
		in.Effort = roster.Lead.Effort
	}

	return in
}
