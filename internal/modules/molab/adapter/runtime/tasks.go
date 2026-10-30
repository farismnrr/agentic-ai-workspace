package runtime

import (
	"context"
	"time"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const allTasks = 1 << 30

// Tasks forgets memory-only MoLab pairings on every task termination path.
type Tasks struct {
	taskapp.TaskStore
	Forget func(string)
}

func (s Tasks) close(id string) {
	if s.Forget != nil {
		s.Forget(id)
	}
}
func (s Tasks) SetStatus(ctx context.Context, id string, status taskdomain.Status, at time.Time) error {
	err := s.TaskStore.SetStatus(ctx, id, status, at)
	if err == nil && (status == taskdomain.StatusDone || status == taskdomain.StatusFailed) {
		s.close(id)
	}
	return err
}
func (s Tasks) Finish(ctx context.Context, id, summary string, proof *taskdomain.Proof, at time.Time) error {
	err := s.TaskStore.Finish(ctx, id, summary, proof, at)
	if err == nil {
		s.close(id)
	}
	return err
}
func (s Tasks) FailUnlessDone(ctx context.Context, id string, at time.Time) (bool, error) {
	changed, err := s.TaskStore.FailUnlessDone(ctx, id, at)
	if changed && err == nil {
		s.close(id)
	}
	return changed, err
}
func (s Tasks) Archive(ctx context.Context, user, id string, at time.Time) error {
	err := s.TaskStore.Archive(ctx, user, id, at)
	if err == nil {
		s.close(id)
	}
	return err
}
func (s Tasks) DeleteOwned(ctx context.Context, user string, ids []string) ([]string, error) {
	deleted, err := s.TaskStore.DeleteOwned(ctx, user, ids)
	for _, id := range deleted {
		s.close(id)
	}
	return deleted, err
}
func (s Tasks) DeleteProject(ctx context.Context, user, project string) error {
	tasks, _ := s.All(ctx, user, project, allTasks)
	err := s.TaskStore.DeleteProject(ctx, user, project)
	if err == nil {
		for _, task := range tasks {
			s.close(task.ID)
		}
	}
	return err
}
func (s Tasks) DeleteAll(ctx context.Context, user string) error {
	tasks, _ := s.All(ctx, user, "", allTasks)
	err := s.TaskStore.DeleteAll(ctx, user)
	if err == nil {
		for _, task := range tasks {
			s.close(task.ID)
		}
	}
	return err
}
