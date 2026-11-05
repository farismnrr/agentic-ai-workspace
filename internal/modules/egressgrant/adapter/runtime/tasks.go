package runtime

import (
	"context"
	"time"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// everyTask is "all of them" for the task store's list calls.
const everyTask = 1 << 30

// egressTasks tells the egress grants the moment a task stops being open: finished, failed, archived or deleted.
// It wraps the task store at the composition root, so the task module knows nothing about grants, and every way a
// task can end is covered because every one of them goes through these calls.
type egressTasks struct {
	taskapp.TaskStore
	ended func(ctx context.Context, taskID string)
}

// Tasks wraps task persistence so ending or deleting a task immediately closes its grant state.
func Tasks(store taskapp.TaskStore, ended func(context.Context, string)) taskapp.TaskStore {
	return egressTasks{TaskStore: store, ended: ended}
}

func (s egressTasks) SetStatus(ctx context.Context, id string, status taskdomain.Status, at time.Time) error {
	if err := s.TaskStore.SetStatus(ctx, id, status, at); err != nil {
		return err //nolint:wrapcheck // a pass-through
	}

	if status == taskdomain.StatusDone || status == taskdomain.StatusFailed {
		s.ended(ctx, id)
	}

	return nil
}

func (s egressTasks) CloseStaleNeedsInput(ctx context.Context, userID, id string, at time.Time) error {
	if err := s.TaskStore.CloseStaleNeedsInput(ctx, userID, id, at); err != nil {
		return err //nolint:wrapcheck // a pass-through
	}
	s.ended(ctx, id)
	return nil
}

func (s egressTasks) Archive(ctx context.Context, userID, id string, at time.Time) error {
	if err := s.TaskStore.Archive(ctx, userID, id, at); err != nil {
		return err //nolint:wrapcheck // a pass-through
	}

	s.ended(ctx, id)

	return nil
}

func (s egressTasks) DeleteOwned(ctx context.Context, userID string, ids []string) ([]string, error) {
	deleted, err := s.TaskStore.DeleteOwned(ctx, userID, ids)
	for _, id := range deleted {
		s.ended(ctx, id)
	}

	return deleted, err //nolint:wrapcheck // a pass-through
}

func (s egressTasks) DeleteProject(ctx context.Context, userID, projectID string) error {
	tasks, _ := s.All(ctx, userID, projectID, everyTask)
	err := s.TaskStore.DeleteProject(ctx, userID, projectID)

	if err == nil {
		s.endAll(ctx, tasks)
	}

	return err //nolint:wrapcheck // a pass-through
}

func (s egressTasks) DeleteAll(ctx context.Context, userID string) error {
	tasks, _ := s.All(ctx, userID, "", everyTask)
	err := s.TaskStore.DeleteAll(ctx, userID)

	if err == nil {
		s.endAll(ctx, tasks)
	}

	return err //nolint:wrapcheck // a pass-through
}

func (s egressTasks) endAll(ctx context.Context, tasks []taskdomain.Task) {
	for _, task := range tasks {
		s.ended(ctx, task.ID)
	}
}

// StopExternalLeadTask forwards the owner stop through the lifecycle wrapper.
func (s egressTasks) StopExternalLeadTask(
	ctx context.Context, userID, taskID, source string, at time.Time,
) error {
	stopper, ok := s.TaskStore.(taskapp.ExternalLeadStopStore)
	if !ok {
		return taskdomain.ErrWrongState
	}
	if err := stopper.StopExternalLeadTask(ctx, userID, taskID, source, at); err != nil {
		return err //nolint:wrapcheck // a pass-through
	}
	s.ended(ctx, taskID)
	return nil
}

// EnableExternalDeepReview forwards the owner-only review-mode mutation
// without bypassing the egress task lifecycle wrapper.
func (s egressTasks) EnableExternalDeepReview(
	ctx context.Context, userID, taskID string, at time.Time,
) error {
	promoter, ok := s.TaskStore.(taskapp.ExternalDeepReviewStore)
	if !ok {
		return taskdomain.ErrWrongState
	}
	return promoter.EnableExternalDeepReview(ctx, userID, taskID, at)
}
