package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestBeginAtProjectCapAbandonsOnlyIdleWork(t *testing.T) {
	works, store, now := newWorksTest(t)
	idleAt := now.Add(-31 * time.Minute)
	var workIDs []string
	for index := range domain.MaxOpenWorksProject {
		workID, taskID := fmt.Sprintf("work-%d", index), fmt.Sprintf("task-%d", index)
		workIDs = append(workIDs, workID)
		store.works[workID] = domain.ExternalWork{
			ID: workID, TaskID: taskID, UserID: "u1", Client: "ChatGPT", ProjectID: "proj",
			State: domain.WorkOpen, UpdatedAt: idleAt,
		}
		store.tasks[taskID] = domain.Task{
			ID: taskID, UserID: "u1", Source: "ChatGPT", ProjectID: "proj", WorkID: workID,
			Status: domain.StatusWorking,
		}
		store.events[taskID] = []domain.Event{{Kind: domain.EventRequest, Work: &domain.WorkRef{ID: workID}}}
	}
	running := store.works[workIDs[0]]
	store.events[running.TaskID] = append(store.events[running.TaskID], domain.Event{
		Kind: domain.EventAction, Work: &domain.WorkRef{ID: running.ID, Phase: "running"},
	})

	created, err := works.Begin(context.Background(), beginInput("new work"))
	if err != nil {
		t.Fatal(err)
	}
	if len(store.works) != domain.MaxOpenWorksProject+1 {
		t.Fatalf("new work was not admitted after idle sweep: %d sessions", len(store.works))
	}
	if store.works[workIDs[0]].State != domain.WorkOpen {
		t.Fatal("work with a running call was abandoned")
	}
	for _, workID := range workIDs[1:] {
		if store.works[workID].State != domain.WorkAbandoned ||
			store.tasks[store.works[workID].TaskID].Status != domain.StatusFailed {
			t.Errorf("idle work %s was not abandoned", workID)
		}
	}
	if store.works[created.WorkID].State != domain.WorkOpen {
		t.Fatal("new work did not remain open")
	}
}

func TestBeginEnforcesUserOpenWorkCapAcrossProjects(t *testing.T) {
	works, store, _ := newWorksTest(t)
	for i := 0; i < domain.MaxOpenWorksUser; i++ {
		project := "proj"
		if i%2 == 1 {
			project = "other"
		}
		id := fmt.Sprintf("seed-%02d", i)
		store.works[id] = domain.ExternalWork{
			ID: id, UserID: "u1", Client: "ChatGPT", ProjectID: project, State: domain.WorkOpen,
			UpdatedAt: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC),
		}
	}
	if _, err := works.Begin(context.Background(), beginInput("over user cap")); !errors.Is(err, domain.ErrTooMany) {
		t.Fatalf("user cap error = %v", err)
	}
}

func TestWaitingForPersonDoesNotBecomeStaleOrAbandoned(t *testing.T) {
	works, store, now := newWorksTest(t)
	created, err := works.Begin(context.Background(), beginInput("Review plan"))
	if err != nil {
		t.Fatal(err)
	}
	work := store.works[created.WorkID]
	work.PlanState = domain.PlanReadyForApproval
	work.UpdatedAt = now.Add(-48 * time.Hour)
	store.works[work.ID] = work
	if abandonErr := works.abandonStale(context.Background(), "u1"); abandonErr != nil {
		t.Fatal(abandonErr)
	}
	state, err := works.State(context.Background(), "u1", "ChatGPT", "proj", work.ID)
	if err != nil || state.Stale || store.works[work.ID].State != domain.WorkOpen {
		t.Fatalf("person review was treated as stale work: state=%+v work=%+v err=%v", state, store.works[work.ID], err)
	}
}
