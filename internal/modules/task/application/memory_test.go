package application_test

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type row struct {
	role  domain.Role
	block domain.Block
}

// memory implements every store port in memory.
type memory struct {
	tasks      map[string]domain.Task
	events     map[string][]domain.Event
	transcript map[string][]row
	usage      []application.UsageRecord
	mu         sync.Mutex
	counter    int
}

func newMemory() *memory {
	return &memory{
		tasks: map[string]domain.Task{}, events: map[string][]domain.Event{}, transcript: map[string][]row{},
	}
}

func (m *memory) ID() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counter++

	return "id" + strconv.Itoa(m.counter), nil
}

func (m *memory) Create(_ context.Context, task domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tasks[task.ID] = task

	return nil
}

func (m *memory) Find(_ context.Context, userID, id string) (domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok || (userID != "" && task.UserID != userID) {
		return domain.Task{}, domain.ErrNotFound
	}

	return task, nil
}

func (m *memory) List(_ context.Context, userID, _ string, _ int) ([]domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var tasks []domain.Task

	for _, task := range m.tasks {
		if task.UserID == userID {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (m *memory) update(id string, change func(*domain.Task)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return domain.ErrNotFound
	}

	change(&task)
	m.tasks[id] = task

	return nil
}

func (m *memory) SetStatus(_ context.Context, id string, status domain.Status, _ time.Time) error {
	return m.update(id, func(t *domain.Task) { t.Status = status })
}

func (m *memory) SetContext(_ context.Context, id string, used, window int64) error {
	return m.update(id, func(t *domain.Task) { t.ContextUsed, t.ContextWindow = used, window })
}

func (m *memory) SavePlan(_ context.Context, id string, plan []domain.PlanStep, _ time.Time) error {
	return m.update(id, func(t *domain.Task) { t.Plan = plan })
}

func (m *memory) SavePRD(_ context.Context, id, prd string, _ time.Time) error {
	return m.update(id, func(t *domain.Task) { t.PRD = prd })
}

func (m *memory) Finish(_ context.Context, id, summary string, proof *domain.Proof, _ time.Time) error {
	return m.update(id, func(t *domain.Task) { t.Status, t.Summary, t.Proof = domain.StatusReview, summary, proof })
}

func (m *memory) DeleteAll(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, task := range m.tasks {
		if task.UserID == userID {
			delete(m.tasks, id)
		}
	}

	return nil
}

func (m *memory) Running(context.Context) ([]domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var tasks []domain.Task

	for _, task := range m.tasks {
		if task.Status.Running() {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}
