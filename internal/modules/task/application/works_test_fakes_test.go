package application

import (
	"context"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type workFakeStore struct {
	works  map[string]domain.ExternalWork
	tasks  map[string]domain.Task
	events map[string][]domain.Event
	ops    map[string]WorkOp
	mu     sync.Mutex
}

func newWorkFakeStore() *workFakeStore {
	return &workFakeStore{
		works:  map[string]domain.ExternalWork{},
		tasks:  map[string]domain.Task{},
		events: map[string][]domain.Event{},
		ops:    map[string]WorkOp{},
	}
}

func (f *workFakeStore) CreateWork(_ context.Context, create WorkCreate) (domain.ExternalWork, error) {
	task, work, event := create.Task, create.Work, create.Event
	f.mu.Lock()
	defer f.mu.Unlock()
	if work.IdemKey != "" {
		for _, prior := range f.works {
			if prior.UserID == work.UserID && prior.Client == work.Client &&
				prior.ProjectID == work.ProjectID && prior.IdemKey == work.IdemKey {
				return prior, nil
			}
		}
	}
	f.tasks[task.ID] = task
	f.works[work.ID] = work
	f.events[task.ID] = []domain.Event{event}
	return work, nil
}

func (f *workFakeStore) Commit(_ context.Context, change WorkChange) ([]domain.Event, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	work, ok := f.works[change.WorkID]
	if !ok || work.UserID != change.UserID {
		return nil, 0, domain.ErrUnknownWork
	}
	if work.Rev != change.ExpectRev {
		return nil, 0, domain.ErrStaleRevision
	}
	applyFakeWorkPatch(&work, change.Patch)
	work.Rev++
	work.UpdatedAt = time.Now()
	f.works[work.ID] = work
	task := f.tasks[work.TaskID]
	applyFakeTaskPatch(&task, change.Task)
	f.tasks[task.ID] = task
	f.events[task.ID] = append(f.events[task.ID], change.Events...)
	if change.Op != nil {
		f.ops[change.WorkID+":"+change.Op.OpKey] = *change.Op
	}
	return change.Events, work.Rev, nil
}

func (f *workFakeStore) FindWork(
	_ context.Context, userID, client, projectID, workID string,
) (domain.ExternalWork, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	work, ok := f.works[workID]
	if !ok || work.UserID != userID || work.Client != client || work.ProjectID != projectID {
		return domain.ExternalWork{}, domain.ErrUnknownWork
	}
	return work, nil
}

func (f *workFakeStore) FindIdempotent(
	_ context.Context, userID, client, projectID, key string,
) (domain.ExternalWork, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, work := range f.works {
		if work.UserID == userID && work.Client == client && work.ProjectID == projectID && work.IdemKey == key {
			return work, true, nil
		}
	}
	return domain.ExternalWork{}, false, nil
}

func (f *workFakeStore) FindOp(_ context.Context, workID, opKey string) (WorkOp, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	op, ok := f.ops[workID+":"+opKey]
	return op, ok, nil
}

func (f *workFakeStore) StaleOpen(_ context.Context, userID string, before time.Time) ([]domain.ExternalWork, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stale := []domain.ExternalWork{}
	for _, work := range f.works {
		if work.UserID == userID && work.State == domain.WorkOpen && !work.UpdatedAt.After(before) {
			stale = append(stale, work)
		}
	}
	return stale, nil
}

func (f *workFakeStore) CountOpen(_ context.Context, userID, projectID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, work := range f.works {
		if work.UserID == userID && work.State == domain.WorkOpen && (projectID == "" || work.ProjectID == projectID) {
			count++
		}
	}
	return count, nil
}

func (f *workFakeStore) ByTask(_ context.Context, taskID string) (domain.ExternalWork, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, work := range f.works {
		if work.TaskID == taskID {
			return work, nil
		}
	}
	return domain.ExternalWork{}, domain.ErrUnknownWork
}

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

var _ WorkStore = (*workFakeStore)(nil)
