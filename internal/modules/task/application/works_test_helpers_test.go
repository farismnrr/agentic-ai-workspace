package application

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func newWorksTest(t *testing.T) (*Works, *workFakeStore, *time.Time) {
	t.Helper()
	store := newWorkFakeStore()
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	deps := testDeps(store, &now)
	return NewWorks(deps, store), store, &now
}

func testDeps(store *workFakeStore, now *time.Time) Deps {
	return Deps{
		Tasks: workTaskFake{store: store}, Events: workEventFake{store: store}, Projects: workProjects{},
		IDs: &workIDs{}, Team: workRoster{}, Now: func() time.Time { return *now },
	}
}

func beginInput(summary string) BeginInput {
	return BeginInput{UserID: "u1", Client: "ChatGPT", ProjectID: "proj", Summary: summary}
}

type workIDs struct {
	mu sync.Mutex
	n  int
}

func (i *workIDs) ID() (string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.n++
	return fmt.Sprintf("id-%d", i.n), nil
}

type workProjects struct{}

func (workProjects) Name(_ context.Context, _, project string) (string, error) {
	if project == "proj" || project == "other" {
		return project, nil
	}
	return "", domain.ErrNoProject
}

type workRoster struct{}

func (workRoster) Roster(context.Context, string) (Roster, error) {
	return Roster{Lead: AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true}, Members: []AgentRole{
		{Key: "developer", Name: "Developer", Purpose: "Writes code.", Prompt: "private prompt"},
	}}, nil
}
