package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestBeginSeparatesSessionsWithinLegacyWindow(t *testing.T) {
	works, store, _ := newWorksTest(t)
	first, err := works.Begin(context.Background(), beginInput("first request"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := works.Begin(context.Background(), beginInput("second request"))
	secondTask := store.works[second.WorkID].TaskID
	firstTask := store.works[first.WorkID].TaskID
	if err != nil || second.WorkID == first.WorkID || secondTask == firstTask {
		t.Fatalf("second begin = %+v, %v", second, err)
	}
	if len(store.works) != 2 {
		t.Fatalf("sessions = %d", len(store.works))
	}
	task := store.tasks[store.works[first.WorkID].TaskID]
	if task.Source != "ChatGPT" || task.Status != domain.StatusWorking {
		t.Fatalf("external task fields = %+v", task)
	}
}

func TestBeginReplaysIdempotentlyAndRedactsSummary(t *testing.T) {
	works, store, _ := newWorksTest(t)
	input := beginInput("token=private-value safe request")
	input.IdempotencyKey = "session-key-001"
	first, err := works.Begin(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := works.Begin(context.Background(), input)
	if err != nil || replay.mode != beginResumed || replay.WorkID != first.WorkID {
		t.Fatalf("replay = %+v, %v", replay, err)
	}
	task := store.tasks[store.works[first.WorkID].TaskID]
	if strings.Contains(task.Summary, "private-value") {
		t.Fatal("secret was saved in the task summary")
	}
	encoded, err := json.Marshal(replay)
	if err != nil || !strings.Contains(string(encoded), `"resumed":true`) ||
		!strings.Contains(string(encoded), `"idempotency":"keyed"`) {
		t.Fatalf("replay response = %s, %v", encoded, err)
	}
}

func TestConcurrentBeginRetriesShareOneWork(t *testing.T) {
	works, store, _ := newWorksTest(t)
	input := beginInput("concurrent request")
	input.IdempotencyKey = "concurrent-key-01"
	start := make(chan struct{})
	results := make(chan BeginResult, 2)
	errorsFound := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			result, err := works.Begin(context.Background(), input)
			if err != nil {
				errorsFound <- err
				return
			}
			results <- result
		}()
	}
	close(start)
	group.Wait()
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
	first := <-results
	second := <-results
	if first.WorkID != second.WorkID || len(store.works) != 1 {
		t.Fatalf("concurrent retries created different works: %+v %+v (%d total)", first, second, len(store.works))
	}
}

func TestBeginWithoutKeyReportsNoIdempotency(t *testing.T) {
	works, _, _ := newWorksTest(t)
	result, err := works.Begin(context.Background(), beginInput("request"))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil || !strings.Contains(string(encoded), `"idempotency":"none"`) {
		t.Fatalf("begin response = %s, %v", encoded, err)
	}
}

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

func TestBeginRejectsReusedKeyWithDifferentRequest(t *testing.T) {
	works, _, _ := newWorksTest(t)
	input := beginInput("first request")
	input.IdempotencyKey = "request-001"
	if _, err := works.Begin(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	input.Summary = "second request"
	if _, err := works.Begin(context.Background(), input); !errors.Is(err, domain.ErrKeyReused) {
		t.Fatalf("mismatched retry error = %v", err)
	}
}

func TestBeginCapsAndRedactsRequest(t *testing.T) {
	works, store, _ := newWorksTest(t)
	for i := 0; i < domain.MaxOpenWorksProject; i++ {
		input := beginInput(fmt.Sprintf("request %d", i))
		input.IdempotencyKey = fmt.Sprintf("request-%03d", i)
		if _, err := works.Begin(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}
	limited, _, _ := newWorksTest(t)
	limited.store = store
	if _, err := limited.Begin(context.Background(), beginInput("another")); !errors.Is(err, domain.ErrTooMany) {
		t.Fatalf("project cap error = %v", err)
	}
	for _, task := range store.tasks {
		if task.Summary != "" && task.Summary == "token=private-value" {
			t.Fatal("secret was saved in the task summary")
		}
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
