package sqlite

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

//nolint:govet // fieldalignment: this mirrors the call result consumed by the assertion helper.
type workCallResult struct {
	event  domain.Event
	taskID string
	err    error
}

//nolint:gocyclo,cyclop // checks role capture and the in-place event update together.
func TestWorkCallsCaptureRoleAndUpdateLongCommandInOneEvent(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-call", "t1", "")
	role, roleName := "developer", "Developer"
	if _, _, err := store.Commit(ctx, application.WorkChange{
		WorkID: work.ID, UserID: "u", ExpectRev: work.Rev,
		Patch: &application.WorkPatch{RoleKey: &role, RoleName: &roleName},
	}); err != nil {
		t.Fatal(err)
	}
	started := domain.Event{
		Kind: domain.EventAction, Text: "Running: go test", Work: &domain.WorkRef{ID: work.ID, Phase: "running"},
	}
	taskID, event, err := store.AppendWorkCall(ctx, "u", "client", "p", work.ID, started)
	if err != nil || taskID != "t1" || event.Work.RoleName != "Developer" {
		t.Fatalf("start event = %+v task=%q err=%v", event, taskID, err)
	}
	finished := domain.Event{
		Kind: domain.EventAction, Text: "go test", Run: &domain.RunInfo{Name: "go test", ExitCode: 3},
		Work: &domain.WorkRef{ID: work.ID, Phase: "finished", Evidence: domain.EvidenceRan},
	}
	if _, updateErr := store.UpdateWorkCall(ctx, taskID, work.ID, event.ID, finished); updateErr != nil {
		t.Fatal(updateErr)
	}
	events, err := store.EventsAfter(ctx, taskID, 0)
	if err != nil || len(events) != 2 {
		t.Fatalf("events = %d, %v", len(events), err)
	}
	got := events[1]
	if got.ID != event.ID || got.Work.RoleName != "Developer" || got.Work.Phase != "finished" ||
		got.Work.Evidence != domain.EvidenceRan || got.Run == nil || got.Run.ExitCode != 3 {
		t.Fatalf("finished event = %+v", got)
	}
}

func TestWorkCallsRejectClosedWork(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-closed", "t1", "")
	state, ended := domain.WorkCompleted, time.Now()
	if _, _, err := store.Commit(ctx, application.WorkChange{
		WorkID: work.ID, UserID: "u", ExpectRev: work.Rev,
		Patch: &application.WorkPatch{State: &state, EndedAt: &ended},
	}); err != nil {
		t.Fatal(err)
	}
	_, _, err := store.AppendWorkCall(ctx, "u", "client", "p", work.ID,
		domain.Event{Kind: domain.EventAction, Text: "edit", Work: &domain.WorkRef{ID: work.ID}})
	if !errors.Is(err, domain.ErrWorkClosed) {
		t.Fatalf("closed work call error = %v", err)
	}
}

func TestWorkCallChangeKeepsOneUndoRecordAndCanBeUndoneOnce(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-edit", "t-edit", "")
	taskID, event, err := store.AppendWorkCall(ctx, "u", "client", "p", work.ID, domain.Event{
		Kind: domain.EventAction, Text: "edit page", Files: []string{"page.ts"}, Undo: []byte("undo-page"),
		Work: &domain.WorkRef{ID: work.ID},
	})
	if err != nil || !event.CanUndo() || len(event.Files) != 1 || string(event.Undo) != "undo-page" {
		t.Fatalf("work change event = %+v, task=%q, err=%v", event, taskID, err)
	}
	event.Undone = true
	if updateErr := store.UpdateEvent(ctx, taskID, event); updateErr != nil {
		t.Fatal(updateErr)
	}
	stored, err := store.FindEvent(ctx, taskID, event.ID)
	if err != nil || stored.CanUndo() {
		t.Fatalf("undone event still offers undo: %+v, %v", stored, err)
	}
}

func TestConcurrentWorkCallsKeepTheirOwnTaskAndRole(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	works := []domain.ExternalWork{
		createTestWork(t, ctx, store, "w-a", "t-a", ""),
		createTestWork(t, ctx, store, "w-b", "t-b", ""),
	}
	setWorkRole(t, ctx, store, works[0], "developer", "Developer")
	setWorkRole(t, ctx, store, works[1], "researcher", "Researcher")

	results := make(chan workCallResult, len(works))
	var concurrent sync.WaitGroup
	for _, work := range works {
		work := work
		concurrent.Add(1)
		go func() {
			defer concurrent.Done()
			taskID, event, err := store.AppendWorkCall(ctx, "u", "client", "p", work.ID, domain.Event{
				Kind: domain.EventAction, Text: work.ID, Work: &domain.WorkRef{ID: work.ID},
			})
			results <- workCallResult{taskID: taskID, event: event, err: err}
		}()
	}
	concurrent.Wait()
	close(results)
	checkWorkCallResults(t, results)
}

func setWorkRole(
	t *testing.T, ctx context.Context, store *Store, work domain.ExternalWork, key, name string,
) {
	t.Helper()
	if _, _, err := store.Commit(ctx, application.WorkChange{
		WorkID: work.ID, UserID: "u", ExpectRev: work.Rev,
		Patch: &application.WorkPatch{RoleKey: &key, RoleName: &name},
	}); err != nil {
		t.Fatal(err)
	}
}

func checkWorkCallResults(t *testing.T, results <-chan workCallResult) {
	t.Helper()
	seen := map[string]bool{}
	for got := range results {
		if got.err != nil {
			t.Fatal(got.err)
		}
		wantRole, suffix := "Developer", "a"
		if got.event.Work.ID == "w-b" {
			wantRole, suffix = "Researcher", "b"
		}
		if got.taskID != "t-"+suffix || got.event.Work.RoleName != wantRole || got.event.Text != got.event.Work.ID {
			t.Fatalf("work call mixed across sessions: %+v", got)
		}
		seen[got.event.Work.ID] = true
	}
	if !seen["w-a"] || !seen["w-b"] {
		t.Fatalf("missing work event: %v", seen)
	}
}
