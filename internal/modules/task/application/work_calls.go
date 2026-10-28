package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

var errWorkCallStorageUnavailable = errors.New("connected-app call storage is unavailable")

// StartCall appends the one visible row for a long-running connected-app command.
func (w *Works) StartCall(
	ctx context.Context, userID, client, projectID, workID string, event domain.Event,
) (string, string, error) {
	conversationKind(&event)
	ensureWorkRef(&event, workID)
	event.Work.Phase = "running"
	return w.appendCall(ctx, userID, client, projectID, workID, event, false)
}

// Validate rejects tool calls once their work session has ended.
func (w *Works) Validate(ctx context.Context, userID, client, projectID, workID string) error {
	_, _, _, err := w.mutable(ctx, userID, client, projectID, workID)
	return err
}

// RecordCall appends one connected-app action or read under the current role.
func (w *Works) RecordCall(
	ctx context.Context, userID, client, projectID, workID string, event domain.Event, reported bool,
) (string, error) {
	conversationKind(&event)
	ensureWorkRef(&event, workID)
	taskID, _, err := w.appendCall(ctx, userID, client, projectID, workID, event, reported)
	return taskID, err
}

func conversationKind(event *domain.Event) {
	if event.Kind != domain.EventRequest && event.Kind != domain.EventMessage {
		event.Kind = domain.EventAction
	}
}

func ensureWorkRef(event *domain.Event, workID string) {
	if event.Work == nil {
		event.Work = &domain.WorkRef{}
	}
	event.Work.ID = workID
}

func (w *Works) appendCall(
	ctx context.Context, userID, client, projectID, workID string, event domain.Event, reported bool,
) (string, string, error) {
	store, err := w.callStore()
	if err != nil {
		return "", "", err
	}
	cleanWorkEvent(&event)
	taskID, stored, err := store.AppendWorkCall(ctx, userID, client, projectID, workID, event, reported)
	if err != nil {
		return "", "", fmt.Errorf("record connected-app call: %w", err)
	}
	w.notify(ctx, domain.Task{ID: taskID, UserID: userID})
	if len(event.Files) > 0 && w.deps.Notify != nil {
		w.deps.Notify.FilesChanged(projectID)
	}
	return taskID, stored.ID, nil
}

// FinishCall updates the start row of a long-running command in place.
func (w *Works) FinishCall(
	ctx context.Context, taskID, workID, eventID string, event domain.Event,
) error {
	store, err := w.callStore()
	if err != nil {
		return err
	}
	cleanWorkEvent(&event)
	if _, updateErr := store.UpdateWorkCall(ctx, taskID, workID, eventID, event); updateErr != nil {
		return fmt.Errorf("finish connected-app call: %w", updateErr)
	}
	task, err := w.deps.Tasks.Find(ctx, "", taskID)
	if err != nil {
		task.ID = taskID
	}
	w.notify(ctx, task)
	return nil
}

func (w *Works) callStore() (WorkCallStore, error) {
	store, ok := w.store.(WorkCallStore)
	if !ok {
		return nil, errWorkCallStorageUnavailable
	}
	return store, nil
}

func cleanWorkEvent(event *domain.Event) {
	event.Text, _ = domain.Redact(event.Text)
	event.Technical, _ = domain.Redact(event.Technical)
	if event.Run != nil {
		event.Run.Name, _ = domain.Redact(event.Run.Name)
	}
}
