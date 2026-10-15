package application

import (
	"context"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Notifier is told when a task moves on or the model streams text.
type Notifier interface {
	Changed(userID, taskID string)
	Delta(taskID, text string)
}

// Observe wraps the stores so that every write wakes the people following the
// task, and lets the runner stream model text through the same notifier.
func Observe(deps Deps, notifier Notifier) Deps {
	owners := &owners{tasks: deps.Tasks, known: map[string]string{}}
	deps.Tasks = observedTasks{TaskStore: deps.Tasks, notify: notifier, owners: owners}
	deps.Events = observedEvents{EventStore: deps.Events, notify: notifier, owners: owners}
	deps.Notify = notifier

	return deps
}

// owners remembers who owns a task so wake-ups reach the right person.
type owners struct {
	tasks TaskStore
	known map[string]string
	mu    sync.Mutex
}

func (o *owners) of(ctx context.Context, taskID string) string {
	o.mu.Lock()
	user, ok := o.known[taskID]
	o.mu.Unlock()

	if ok {
		return user
	}

	task, err := o.tasks.Find(ctx, "", taskID)
	if err != nil {
		return ""
	}

	o.mu.Lock()
	o.known[taskID] = task.UserID
	o.mu.Unlock()

	return task.UserID
}

type observedTasks struct {
	TaskStore
	notify Notifier
	owners *owners
}

func (o observedTasks) Create(ctx context.Context, task domain.Task) error {
	err := o.TaskStore.Create(ctx, task)
	if err == nil {
		o.notify.Changed(task.UserID, task.ID)
	}

	return err //nolint:wrapcheck // a transparent decorator
}

func (o observedTasks) SetStatus(ctx context.Context, id string, status domain.Status, at time.Time) error {
	err := o.TaskStore.SetStatus(ctx, id, status, at)
	if err == nil {
		o.notify.Changed(o.owners.of(ctx, id), id)
	}

	return err //nolint:wrapcheck // a transparent decorator
}

func (o observedTasks) SavePlan(ctx context.Context, id string, plan []domain.PlanStep, at time.Time) error {
	err := o.TaskStore.SavePlan(ctx, id, plan, at)
	if err == nil {
		o.notify.Changed(o.owners.of(ctx, id), id)
	}

	return err //nolint:wrapcheck // a transparent decorator
}

func (o observedTasks) Finish(ctx context.Context, id, summary string, proof *domain.Proof, at time.Time) error {
	err := o.TaskStore.Finish(ctx, id, summary, proof, at)
	if err == nil {
		o.notify.Changed(o.owners.of(ctx, id), id)
	}

	return err //nolint:wrapcheck // a transparent decorator
}

func (o observedTasks) SetContext(ctx context.Context, id string, used, window int64) error {
	err := o.TaskStore.SetContext(ctx, id, used, window)
	if err == nil {
		o.notify.Changed(o.owners.of(ctx, id), id)
	}

	return err //nolint:wrapcheck // a transparent decorator
}

type observedEvents struct {
	EventStore
	notify Notifier
	owners *owners
}

func (o observedEvents) AppendEvent(ctx context.Context, taskID string, event domain.Event) (domain.Event, error) {
	stored, err := o.EventStore.AppendEvent(ctx, taskID, event)
	if err == nil {
		o.notify.Changed(o.owners.of(ctx, taskID), taskID)
	}

	return stored, err //nolint:wrapcheck // a transparent decorator
}

func (o observedEvents) UpdateEvent(ctx context.Context, taskID string, event domain.Event) error {
	err := o.EventStore.UpdateEvent(ctx, taskID, event)
	if err == nil {
		o.notify.Changed(o.owners.of(ctx, taskID), taskID)
	}

	return err //nolint:wrapcheck // a transparent decorator
}
