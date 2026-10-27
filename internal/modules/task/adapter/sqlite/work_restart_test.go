package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

//nolint:gocyclo,cyclop // the test setup and assertions document the complete persistence restart boundary.
func TestRestartKeepsRunningWorkCommandAndState(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "restart.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, schemaErr := db.ExecContext(ctx, testWorkSchema); schemaErr != nil {
		t.Fatal(schemaErr)
	}
	store := NewStore(db, &workTestIDs{})
	fixtureSQL := `INSERT INTO users(id) VALUES ('u'); INSERT INTO projects(id,user_id) VALUES ('p','u')`
	if _, insertErr := db.ExecContext(ctx, fixtureSQL); insertErr != nil {
		t.Fatal(insertErr)
	}
	createTaskFixture(t, ctx, store, "restart-task", "u", "p")
	work := createTestWork(t, ctx, store, "restart-work", "restart-task", "")
	started := runningCommandEvent(work.ID)
	if _, _, appendErr := store.AppendWorkCall(ctx, "u", "client", "p", work.ID, started); appendErr != nil {
		t.Fatal(appendErr)
	}
	if closeErr := db.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	reopenedDB, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopenedDB.Close() })
	reopened := NewStore(reopenedDB, &workTestIDs{})
	now := func() time.Time { return time.Unix(2, 0) }
	service := application.NewService(application.Deps{Tasks: reopened, Events: reopened, Now: now}, restartDispatcher{})
	if recoverErr := service.Recover(ctx); recoverErr != nil {
		t.Fatal(recoverErr)
	}
	works := application.NewWorks(application.Deps{Tasks: reopened, Events: reopened, Now: now}, reopened)
	state, err := works.State(ctx, "u", "client", "p", work.ID)
	if err != nil {
		t.Fatal(err)
	}
	events, err := reopened.EventsAfter(ctx, work.TaskID, 0)
	if err != nil || len(events) != 2 || events[1].Work == nil || events[1].Work.Phase != "running" {
		t.Fatalf("persisted running command = %+v, %v", events, err)
	}
	if state.WorkID != work.ID || state.State != domain.WorkOpen || state.Revision != work.Rev ||
		state.TaskID != work.TaskID {
		t.Fatalf("reopened state = %+v, want work %+v", state, work)
	}
}

func runningCommandEvent(workID string) domain.Event {
	return domain.Event{
		Kind: domain.EventAction, Text: "Running: go test",
		Work: &domain.WorkRef{ID: workID, Phase: "running"},
	}
}

type restartDispatcher struct{}

func (restartDispatcher) Enqueue(string, string)                {}
func (restartDispatcher) Cancel(string) application.CancelState { return application.CancelNone }
