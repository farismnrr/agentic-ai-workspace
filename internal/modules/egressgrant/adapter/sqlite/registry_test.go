package sqlite_test

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

var errRegistry = errors.New("registry refused")

// memRegistry stands in for the proxy's grant table: ids to the host and end they were allowed for.
type memRegistry struct {
	held map[string]string
	fail error
	mu   sync.Mutex
}

func (m *memRegistry) Allow(id, projectID, taskID, host string, ends time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.fail != nil {
		return m.fail
	}

	if _, ok := m.held[id]; !ok {
		m.held[id] = projectID + "/" + taskID + "/" + host + "/" + ends.UTC().Format(time.RFC3339)
	}

	return nil
}

func (m *memRegistry) Revoke(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.held, id)
}

func (m *memRegistry) IDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]string, 0, len(m.held))
	for id := range m.held {
		ids = append(ids, id)
	}

	slices.Sort(ids)

	return ids
}

func (m *memRegistry) EndTask(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, v := range m.held {
		if strings.Split(v, "/")[1] == taskID {
			delete(m.held, id)
		}
	}
}

func (m *memRegistry) has(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.held[id]

	return ok
}

// enforced is a world whose service keeps a registry in line with the database.
func enforced(t *testing.T) (world, *memRegistry) {
	t.Helper()

	w := newWorld(t)
	reg := &memRegistry{held: map[string]string{}}
	w.WithRegistry(reg)

	return w, reg
}
