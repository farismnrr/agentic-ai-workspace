package application_test

import (
	"context"
	"strconv"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (m *memory) AppendEvent(_ context.Context, taskID string, event domain.Event) (domain.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counter++
	event.ID = "e" + strconv.Itoa(m.counter)
	event.Seq = int64(len(m.events[taskID]) + 1)
	m.events[taskID] = append(m.events[taskID], event)

	return event, nil
}

func (m *memory) EventsAfter(_ context.Context, taskID string, seq int64) ([]domain.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var out []domain.Event

	for _, event := range m.events[taskID] {
		if event.Seq > seq {
			out = append(out, event)
		}
	}

	return out, nil
}

func (m *memory) FindEvent(_ context.Context, taskID, id string) (domain.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, event := range m.events[taskID] {
		if event.ID == id {
			return event, nil
		}
	}

	return domain.Event{}, domain.ErrNotFound
}

func (m *memory) EventByToolUse(_ context.Context, taskID, toolUseID string) (domain.Event, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, event := range m.events[taskID] {
		if event.ToolUseID == toolUseID && toolUseID != "" {
			return event, true, nil
		}
	}

	return domain.Event{}, false, nil
}

func (m *memory) UpdateEvent(_ context.Context, taskID string, updated domain.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, event := range m.events[taskID] {
		if event.ID == updated.ID {
			m.events[taskID][i] = updated

			return nil
		}
	}

	return domain.ErrNotFound
}

func (m *memory) AppendBlock(_ context.Context, taskID string, role domain.Role, block domain.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transcript[taskID] = append(m.transcript[taskID], row{role: role, block: block})

	return nil
}

func (m *memory) LoadTranscript(_ context.Context, taskID string) ([]domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var messages []domain.Message

	for _, r := range m.transcript[taskID] {
		if n := len(messages); n > 0 && messages[n-1].Role == r.role {
			messages[n-1].Blocks = append(messages[n-1].Blocks, r.block)

			continue
		}

		messages = append(messages, domain.Message{Role: r.role, Blocks: []domain.Block{r.block}})
	}

	return messages, nil
}

func (m *memory) AddUsage(_ context.Context, record application.UsageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.usage = append(m.usage, record)

	return nil
}

func (m *memory) TotalUsage(_ context.Context, taskID string) (domain.Usage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var total domain.Usage

	for _, record := range m.usage {
		if record.TaskID == taskID {
			total.Input += record.Usage.Input
			total.Output += record.Usage.Output
		}
	}

	return total, nil
}

func (m *memory) UsageSummary(context.Context, string, time.Time) ([]application.UsageRow, error) {
	return nil, nil
}
