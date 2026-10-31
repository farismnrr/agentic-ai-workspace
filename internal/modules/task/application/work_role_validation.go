package application

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func publicRoles(roster Roster) []WorkRole {
	roles := make([]WorkRole, 0, len(roster.Members))
	for _, member := range roster.Members {
		if !member.Lead {
			roles = append(roles, WorkRole{Key: member.Key, Name: member.Name, Purpose: member.Purpose})
		}
	}
	return roles
}

func normalizePlanned(names []string, roster Roster) ([]string, error) {
	roles := publicRoles(roster)
	planned := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		role, ok := matchRole(name, roster, roles)
		if !ok {
			return nil, badRole(roster)
		}
		if !seen[role.Key] {
			planned = append(planned, role.Key)
			seen[role.Key] = true
		}
	}
	return planned, nil
}

func availableRoleInRoster(name string, roster Roster) (WorkRole, error) {
	role, ok := matchRole(name, roster, publicRoles(roster))
	if !ok {
		return WorkRole{}, badRole(roster)
	}
	return role, nil
}

func matchRole(name string, roster Roster, roles []WorkRole) (WorkRole, bool) {
	needle := canonicalMemberKey(strings.TrimSpace(name))
	if needle == "" || strings.EqualFold(needle, "orchestrator") ||
		(strings.EqualFold(needle, roster.Lead.Key) && roster.Lead.Key != "") ||
		(strings.EqualFold(needle, roster.Lead.Name) && roster.Lead.Name != "") {
		return WorkRole{}, false
	}
	for _, role := range roles {
		if strings.EqualFold(needle, role.Key) || strings.EqualFold(needle, role.Name) {
			return role, true
		}
	}
	return WorkRole{}, false
}

func badRole(roster Roster) error {
	keys := make([]string, 0, len(roster.Members))
	for _, role := range publicRoles(roster) {
		keys = append(keys, role.Key)
	}
	sort.Strings(keys)
	allowed := "none"
	if len(keys) > 0 {
		allowed = strings.Join(keys, ", ")
	}
	return fmt.Errorf("%w: allowed role keys: %s", domain.ErrBadRole, allowed)
}

// AvailableRoles returns the enabled non-lead roles for a connected app response.
func (w *Works) AvailableRoles(ctx context.Context, userID string) ([]WorkRole, error) {
	roster, err := w.roster(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load work roles: %w", err)
	}
	return publicRoles(roster), nil
}
