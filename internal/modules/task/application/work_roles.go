package application

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

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

// SetRole changes work attribution only. It does not participate in access checks.
func (w *Works) SetRole(ctx context.Context, userID, client, projectID, workID, roleName string) (WorkRole, error) {
	return w.SetRoleWithBrief(ctx, userID, client, projectID, workID, roleName, "")
}

// SetRoleWithBrief records a bounded explanation with the attribution change; it does not alter permissions.
func (w *Works) SetRoleWithBrief(
	ctx context.Context, userID, client, projectID, workID, roleName, brief string,
) (WorkRole, error) {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return WorkRole{}, err
	}
	cleanBrief, err := sanitizeText(brief, domain.MaxWorkClaim)
	if err != nil {
		return WorkRole{}, err
	}
	work, err := w.store.FindWork(ctx, userID, client, projectID, workID)
	if err != nil {
		return WorkRole{}, domain.ErrUnknownWork
	}
	if work.State != domain.WorkOpen && work.State != domain.WorkBlocked {
		return WorkRole{}, domain.ErrWorkClosed
	}
	role, err := w.availableRole(ctx, userID, roleName)
	if err != nil {
		return WorkRole{}, err
	}
	if roleAlreadySet(work, role, cleanBrief) {
		return role, nil
	}
	change := roleChange(work, userID, role, cleanBrief, w.now())
	_, _, err = w.store.Commit(ctx, change)
	if err != nil {
		return WorkRole{}, fmt.Errorf("set work role: %w", err)
	}
	w.notify(ctx, domain.Task{ID: work.TaskID, UserID: userID})
	return role, nil
}

func roleAlreadySet(work domain.ExternalWork, role WorkRole, brief string) bool {
	return work.RoleKey == role.Key && work.State == domain.WorkOpen && brief == ""
}

func (w *Works) availableRole(ctx context.Context, userID, name string) (WorkRole, error) {
	if utf8.RuneCountInString(name) > domain.MaxWorkClaim {
		return WorkRole{}, domain.ErrBadRequest
	}
	roster, err := w.roster(ctx, userID)
	if err != nil {
		return WorkRole{}, fmt.Errorf("load work roles: %w", err)
	}
	return availableRoleInRoster(name, roster)
}

func roleChange(work domain.ExternalWork, userID string, role WorkRole, brief string, at time.Time) WorkChange {
	patch, taskPatch := rolePatches(work, role)
	events := roleEvents(work, role, brief, at)
	return WorkChange{
		WorkID: work.ID, UserID: userID, ExpectRev: work.Rev,
		Patch: patch, Task: taskPatch, Events: events,
	}
}

func rolePatches(work domain.ExternalWork, role WorkRole) (*WorkPatch, *TaskPatch) {
	patch := &WorkPatch{}
	var taskPatch *TaskPatch
	if work.RoleKey != role.Key {
		patch.RoleKey, patch.RoleName = &role.Key, &role.Name
	}
	if work.State == domain.WorkBlocked {
		patch.State = workStatePtr(domain.WorkOpen)
		taskPatch = &TaskPatch{Status: statusPtr(domain.StatusWorking)}
	}
	return patch, taskPatch
}

func roleEvents(work domain.ExternalWork, role WorkRole, brief string, at time.Time) []domain.Event {
	events := []domain.Event{}
	if work.State == domain.WorkBlocked {
		events = append(events, resumedEvent(work, at))
	}
	if work.RoleKey != role.Key || brief != "" {
		events = append(events, roleEvent(work, role, brief, at))
	}
	return events
}

func roleEvent(work domain.ExternalWork, role WorkRole, brief string, at time.Time) domain.Event {
	return domain.Event{
		Kind: domain.EventRole, Text: role.Name, Technical: brief, Time: at,
		Work: &domain.WorkRef{ID: work.ID, Role: role.Key, RoleName: role.Name, From: work.RoleKey},
	}
}
