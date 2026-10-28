package application

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (w *Works) mutable(ctx context.Context, userID, client, projectID, workID string) (
	domain.ExternalWork, domain.Task, bool, error,
) {
	work, err := w.store.FindWork(ctx, userID, client, projectID, workID)
	if err != nil {
		return domain.ExternalWork{}, domain.Task{}, false, domain.ErrUnknownWork
	}
	if work.State != domain.WorkOpen && work.State != domain.WorkBlocked {
		return domain.ExternalWork{}, domain.Task{}, false, domain.ErrWorkClosed
	}
	task, err := w.deps.Tasks.Find(ctx, userID, work.TaskID)
	if err != nil || task.ProjectID != projectID || task.Source != client || task.WorkID != work.ID {
		return domain.ExternalWork{}, domain.Task{}, false, domain.ErrUnknownWork
	}
	return work, task, work.State == domain.WorkBlocked, nil
}

func workEvent(work domain.ExternalWork, kind domain.EventKind, text, technical string, at time.Time) domain.Event {
	return domain.Event{
		Kind: kind, Text: text, Technical: technical, Time: at,
		Work: &domain.WorkRef{ID: work.ID, Actor: work.Client, Role: work.RoleKey, RoleName: work.RoleName},
	}
}

func resumedEvent(work domain.ExternalWork, at time.Time) domain.Event {
	event := workEvent(work, domain.EventMessage, "Work resumed.", "", at)
	event.Work.Phase = "resumed"
	return event
}

func workStatePtr(state domain.WorkState) *domain.WorkState { return &state }

func (w *Works) resumedChange(work domain.ExternalWork, task *TaskPatch, patch *WorkPatch,
	events []domain.Event, at time.Time,
) (*TaskPatch, *WorkPatch, []domain.Event) {
	if work.State != domain.WorkBlocked {
		return task, patch, events
	}
	if task == nil {
		task = &TaskPatch{}
	}
	if patch == nil {
		patch = &WorkPatch{}
	}
	if task.Status == nil {
		task.Status = statusPtr(domain.StatusWorking)
	}
	if patch.State == nil {
		patch.State = workStatePtr(domain.WorkOpen)
	}
	return task, patch, append([]domain.Event{resumedEvent(work, at)}, events...)
}

func sanitizeText(value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > max {
		return "", domain.ErrTooLarge
	}
	clean, _ := domain.Redact(value)
	if utf8.RuneCountInString(clean) > max {
		return "", domain.ErrTooLarge
	}
	return clean, nil
}

func (w *Works) syncWorkNotes(ctx context.Context, work domain.ExternalWork) {
	if w.deps.Notes == nil {
		return
	}
	task, err := w.deps.Tasks.Find(ctx, "", work.TaskID)
	if err != nil {
		return
	}
	_, err = NewService(w.deps, nil).projectDocuments(ctx, task)
	if err == nil && w.deps.Notify != nil {
		w.deps.Notify.FilesChanged(task.ProjectID)
	}
}

func (w *Works) appendMutation(ctx context.Context, work domain.ExternalWork, task *TaskPatch,
	patch *WorkPatch, events []domain.Event,
) error {
	task, patch, events = w.resumedChange(work, task, patch, events, w.now())
	if patch == nil {
		patch = &WorkPatch{}
	}
	if patch.State == nil || *patch.State == domain.WorkOpen || *patch.State == domain.WorkBlocked {
		patch.ClearReportedAt = true
	}
	_, _, err := w.store.Commit(ctx, WorkChange{
		WorkID: work.ID, UserID: work.UserID,
		ExpectRev: work.Rev, Task: task, Patch: patch, Events: events,
	})
	if err != nil {
		return fmt.Errorf("save work update: %w", err)
	}
	w.syncWorkNotes(ctx, work)
	w.notify(ctx, domain.Task{ID: work.TaskID, UserID: work.UserID})
	return nil
}
