package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type workTaskFake struct {
	TaskStore
	store *workFakeStore
}

func (f workTaskFake) Find(_ context.Context, userID, taskID string) (domain.Task, error) {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()
	task, ok := f.store.tasks[taskID]
	if !ok || task.UserID != userID {
		return domain.Task{}, domain.ErrNotFound
	}
	return task, nil
}

type workEventFake struct {
	EventStore
	store *workFakeStore
}

func (f workEventFake) EventsAfter(_ context.Context, taskID string, _ int64) ([]domain.Event, error) {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()
	return append([]domain.Event{}, f.store.events[taskID]...), nil
}

func (f workEventFake) EventsChangedAfter(_ context.Context, taskID string, version int64) ([]domain.Event, error) {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()
	var events []domain.Event
	for _, event := range f.store.events[taskID] {
		if event.Version > version {
			events = append(events, event)
		}
	}
	return events, nil
}

var _ WorkStore = (*workFakeStore)(nil)
