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
	if err != nil || roster.Lead.ProviderID == "" || roster.Lead.Model == "" {
		return in
	}

	in.ProviderID, in.Model = roster.Lead.ProviderID, roster.Lead.Model
	if in.Effort == "" {
		in.Effort = roster.Lead.Effort
	}

	return in
}
