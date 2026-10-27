package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (w *Works) roster(ctx context.Context, userID string) (Roster, error) {
	if w.deps.Team == nil {
		return Roster{}, nil
	}
	roster, err := w.deps.Team.Roster(ctx, userID)
	if err != nil {
		return Roster{}, fmt.Errorf("load team roster: %w", err)
	}
	return roster, nil
}

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
			return nil, domain.ErrBadRole
		}
		if !seen[role.Key] {
			planned = append(planned, role.Key)
			seen[role.Key] = true
		}
	}
	return planned, nil
}

func matchRole(name string, roster Roster, roles []WorkRole) (WorkRole, bool) {
	needle := strings.TrimSpace(name)
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

// SetRole changes work attribution only. It does not participate in access checks.
func (w *Works) SetRole(ctx context.Context, userID, client, projectID, workID, roleName string) (WorkRole, error) {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return WorkRole{}, err
	}
	work, err := w.store.FindWork(ctx, userID, client, projectID, workID)
	if err != nil {
		return WorkRole{}, domain.ErrUnknownWork
	}
	if work.State != domain.WorkOpen {
		return WorkRole{}, domain.ErrWorkClosed
	}
	roster, err := w.roster(ctx, userID)
	if err != nil {
		return WorkRole{}, fmt.Errorf("load work roles: %w", err)
	}
	role, ok := matchRole(roleName, roster, publicRoles(roster))
	if !ok {
		return WorkRole{}, domain.ErrBadRole
	}
	if work.RoleKey == role.Key {
		return role, nil
	}
	roleKey, roleNameValue := role.Key, role.Name
	_, _, err = w.store.Commit(ctx, WorkChange{
		WorkID: work.ID, UserID: userID, ExpectRev: work.Rev,
		Patch:  &WorkPatch{RoleKey: &roleKey, RoleName: &roleNameValue},
		Events: []domain.Event{roleEvent(work, role, w.now())},
	})
	if err != nil {
		return WorkRole{}, fmt.Errorf("set work role: %w", err)
	}
	w.notify(ctx, domain.Task{ID: work.TaskID, UserID: userID})
	return role, nil
}

func roleEvent(work domain.ExternalWork, role WorkRole, at time.Time) domain.Event {
	return domain.Event{
		Kind: domain.EventRole, Text: role.Name, Time: at,
		Work: &domain.WorkRef{ID: work.ID, Role: role.Key, RoleName: role.Name, From: work.RoleKey},
	}
}
