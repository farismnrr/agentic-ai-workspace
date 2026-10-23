package application_test

import (
	"context"
	"sort"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (m *memory) List(_ context.Context, userID, projectID string, _ int) ([]domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var tasks []domain.Task
	for _, task := range m.tasks {
		if task.UserID == userID && task.ArchivedAt == nil && (projectID == "" || task.ProjectID == projectID) {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *memory) All(_ context.Context, userID, projectID string, _ int) ([]domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var tasks []domain.Task
	for _, task := range m.tasks {
		if task.UserID == userID && (projectID == "" || task.ProjectID == projectID) {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *memory) Archived(_ context.Context, userID string, _ int) ([]domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var tasks []domain.Task
	for _, task := range m.tasks {
		if task.UserID == userID && task.ArchivedAt != nil {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *memory) ListOrchestrator(ctx context.Context, userID, projectID string, limit int) ([]domain.Task, error) {
	tasks, err := m.List(ctx, userID, projectID, limit)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Team && task.Source == "" {
			filtered = append(filtered, task)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
	})
	if limit >= 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (m *memory) Archive(_ context.Context, userID, id string, at time.Time) error {
	return m.updateOwned(userID, id, func(task *domain.Task) {
		if task.ArchivedAt == nil {
			task.ArchivedAt = &at
		}
	})
}

func (m *memory) Restore(_ context.Context, userID, id string) error {
	return m.updateOwned(userID, id, func(task *domain.Task) { task.ArchivedAt = nil })
}

func (m *memory) updateOwned(userID, id string, change func(*domain.Task)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.tasks[id]
	if !ok || task.UserID != userID {
		return domain.ErrNotFound
	}
	change(&task)
	m.tasks[id] = task
	return nil
}

func (m *memory) DeleteOwned(_ context.Context, userID string, ids []string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	deleted := make([]string, 0, len(ids))
	for _, id := range ids {
		if task, ok := m.tasks[id]; ok && task.UserID == userID {
			delete(m.tasks, id)
			deleted = append(deleted, id)
		}
	}
	return deleted, nil
}
