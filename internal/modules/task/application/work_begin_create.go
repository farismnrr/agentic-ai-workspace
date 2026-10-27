package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (w *Works) create(
	ctx context.Context, in BeginInput, summary string, role WorkRole, hash string,
	planned []string, roster Roster,
) (BeginResult, error) {
	taskID, err := w.deps.IDs.ID()
	if err != nil {
		return BeginResult{}, fmt.Errorf("new external task id: %w", err)
	}
	workID, err := w.deps.IDs.ID()
	if err != nil {
		return BeginResult{}, fmt.Errorf("new work id: %w", err)
	}
	now := w.now()
	task := domain.Task{
		ID: taskID, UserID: in.UserID, ProjectID: in.ProjectID, Source: in.Client,
		Title: titleOf(summary), Summary: summary, Status: domain.StatusWorking,
		WorkID: workID, CreatedAt: now, UpdatedAt: now, Plan: []domain.PlanStep{},
	}
	work := domain.ExternalWork{
		ID: workID, TaskID: taskID, UserID: in.UserID, ProjectID: in.ProjectID, Client: in.Client,
		State: domain.WorkOpen, PlanState: domain.PlanDraft, RoleKey: role.Key, RoleName: role.Name,
		IdemKey: in.IdempotencyKey, IdemHash: hash, CreatedAt: now, UpdatedAt: now, NextStep: 1,
	}
	request := domain.Event{
		Kind: domain.EventRequest, Text: summary, Time: now,
		Work: &domain.WorkRef{ID: workID, Actor: in.Client, PlannedRoles: planned},
	}
	initialRole := roleEvent(work, role, "", now)
	initialRole.Work.From = ""
	created, err := w.store.CreateWork(ctx, WorkCreate{
		Task: task, Work: work, Events: []domain.Event{request, initialRole},
	})
	if err != nil {
		return BeginResult{}, fmt.Errorf("create external work: %w", err)
	}
	if created.ID != workID {
		if created.IdemHash != hash {
			return BeginResult{}, domain.ErrKeyReused
		}
		return w.beginResult(created, true, roster), nil
	}
	w.notify(ctx, task)
	return w.beginResult(created, false, roster), nil
}

func beginHash(summary, role string, planned []string) (string, error) {
	data, err := json.Marshal(struct {
		Summary string
		Role    string
		Planned []string
	}{summary, role, planned})
	if err != nil {
		return "", fmt.Errorf("encode begin request: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func validKey(key string) bool {
	if key == "" {
		return true
	}
	length := utf8.RuneCountInString(key)
	return length >= domain.MinRequestID && length <= domain.MaxRequestID
}
