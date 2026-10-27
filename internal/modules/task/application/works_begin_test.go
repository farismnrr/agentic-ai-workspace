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

func TestBeginNotifiesTheUserTaskListAfterCommit(t *testing.T) {
	store := newWorkFakeStore()
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	hub := NewHub()
	defer hub.Close()
	sub := hub.Subscribe(UserTopic("u1"))
	defer sub.Close()
	deps := testDeps(store, &now)
	deps.Notify = hub
	works := NewWorks(deps, store)

	result, err := works.Begin(context.Background(), beginInput("show up in the task list"))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case update := <-sub.C:
		if update.TaskID != store.works[result.WorkID].TaskID {
			t.Fatalf("list notification = %+v", update)
		}
	case <-time.After(time.Second):
		t.Fatal("begin_work did not notify the user's task list")
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
