package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestBeginRequiresEnabledNonLeadRole(t *testing.T) {
	for _, tc := range []struct{ name, role string }{
		{name: "missing"},
		{name: "unknown", role: "analyst"},
		{name: "disabled", role: "designer"},
		{name: "lead key", role: "orchestrator"},
		{name: "lead name", role: "Orchestrator"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			works, store, _ := newWorksTest(t)
			input := beginInput("tracked request")
			input.Role = tc.role
			_, err := works.Begin(context.Background(), input)
			if !errors.Is(err, domain.ErrBadRole) || !strings.Contains(err.Error(),
				"allowed role keys: developer, researcher") {
				t.Fatalf("invalid role error = %v", err)
			}
			if len(store.tasks) != 0 || len(store.works) != 0 {
				t.Fatal("invalid role created executable work")
			}
		})
	}
}

func TestBeginPersistsInitialRoleAndRoleEvent(t *testing.T) {
	works, store, _ := newWorksTest(t)
	result, err := works.Begin(context.Background(), beginInput("tracked request"))
	if err != nil {
		t.Fatal(err)
	}
	work := store.works[result.WorkID]
	events := store.events[work.TaskID]
	if work.RoleKey != "developer" || work.RoleName != "Developer" || len(events) != 2 {
		t.Fatalf("initial role/work events = %+v / %+v", work, events)
	}
	roleEvent := events[1]
	if roleEvent.Kind != domain.EventRole || roleEvent.Work == nil ||
		roleEvent.Work.Role != "developer" || roleEvent.Work.From != "" {
		t.Fatalf("initial role event = %+v", roleEvent)
	}
}

func TestChangingRoleOnlyChangesAttribution(t *testing.T) {
	works, store, _ := newWorksTest(t)
	result, err := works.Begin(context.Background(), beginInput("tracked request"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := works.SetRole(context.Background(), "u1", "ChatGPT", "proj", result.WorkID, "researcher"); err != nil {
		t.Fatal(err)
	}
	work := store.works[result.WorkID]
	if work.RoleKey != "researcher" || store.tasks[work.TaskID].Status != domain.StatusWorking {
		t.Fatalf("role transition altered work execution state: work=%+v task=%+v", work, store.tasks[work.TaskID])
	}
	for _, event := range store.events[work.TaskID] {
		if event.Kind == domain.EventHandoff {
			t.Fatal("role change was incorrectly recorded as a handoff")
		}
	}
}
