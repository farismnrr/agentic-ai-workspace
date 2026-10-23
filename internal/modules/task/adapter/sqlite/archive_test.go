package sqlite

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
)

func TestArchivedListExcludesTaskFromActiveTasks(t *testing.T) {
	ctx, store := newArchiveStore(t)
	id := "000000000000000000000001"
	createArchivedFixture(t, ctx, store, id)
	if err := store.Archive(ctx, "u", id, time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	active, err := store.List(ctx, "u", "", 200)
	if err != nil || len(active) != 0 {
		t.Fatalf("active list contains archived task: %d, %v", len(active), err)
	}
	archived, err := store.Archived(ctx, "u", 10_000)
	if err != nil || len(archived) != 1 || archived[0].ArchivedAt == nil {
		t.Fatalf("archived list: %d, %v", len(archived), err)
	}
	deleted, err := store.DeleteOwned(ctx, "u", []string{id})
	if err != nil || len(deleted) != 1 || deleted[0] != id {
		t.Fatalf("deleted ids: %v, error: %v", deleted, err)
	}
}

func TestOrchestratorListFiltersOwnerProjectModeAndCapsNewestFirst(t *testing.T) {
	ctx, store := newArchiveStore(t)
	_, setupErr := store.db.ExecContext(ctx, `INSERT INTO users (id,name,email,password_hash,created_at)
		VALUES ('other','Other','other@example.test','hash',1);
		INSERT INTO projects (id,user_id,name,slug,created_at) VALUES ('other-p','other','Other project','other-p',1)`)
	if setupErr != nil {
		t.Fatal(setupErr)
	}
	for i := 0; i < 105; i++ {
		id := fmt.Sprintf("%024x", i+1)
		task := domain.Task{
			ID: id, ProjectID: "p", UserID: "u", Title: fmt.Sprintf("Task %03d", i),
			Status: domain.StatusDone, ProviderID: "provider", Model: "model", Team: true,
			CreatedAt: time.Unix(int64(i+1), 0), UpdatedAt: time.Unix(int64(i+1), 0),
		}
		if err := store.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	createModeFixture(t, ctx, store, "solo", "u", "p", false, "", 200)
	createModeFixture(t, ctx, store, "source", "u", "p", true, "connected", 201)
	createModeFixture(t, ctx, store, "foreign", "other", "other-p", true, "", 202)
	createModeFixture(t, ctx, store, "foreign-owner", "other", "p", true, "", 205)
	createModeFixture(t, ctx, store, "archived", "u", "p", true, "", 203)
	if err := store.Archive(ctx, "u", "archived", time.Unix(204, 0)); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListOrchestrator(ctx, "u", "p", 100)
	if err != nil || len(got) != 100 {
		t.Fatalf("got %d tasks, error %v; want capped 100", len(got), err)
	}
	if got[0].Title != "Task 104" || got[len(got)-1].Title != "Task 005" {
		t.Fatalf("task list is not newest-first or cap is wrong: first=%q last=%q", got[0].Title, got[len(got)-1].Title)
	}
}

func createModeFixture(
	t *testing.T, ctx context.Context, store *Store, id, owner, project string, team bool, source string, at int64,
) {
	t.Helper()
	err := store.Create(ctx, domain.Task{
		ID: id, ProjectID: project, UserID: owner, Title: id, Status: domain.StatusDone,
		ProviderID: "provider", Model: "model", Team: team, Source: source,
		CreatedAt: time.Unix(at, 0), UpdatedAt: time.Unix(at, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteOwnedProcessesMoreThan200Rows(t *testing.T) {
	ctx, store := newArchiveStore(t)
	ids := make([]string, 1101)
	for i := range ids {
		ids[i] = fmt.Sprintf("%024x", i+1)
		createArchivedFixture(t, ctx, store, ids[i])
	}
	if deleted, err := store.DeleteOwned(ctx, "u", ids); err != nil || len(deleted) != len(ids) {
		t.Fatalf("deleted %d ids, error: %v", len(deleted), err)
	}
	remaining, err := store.All(ctx, "u", "", -1)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("chunked delete left %d rows: %v", len(remaining), err)
	}
}

func TestArchiveAndRestoreRequireOwnershipAndKeepPin(t *testing.T) {
	ctx, store := newArchiveStore(t)
	id := "000000000000000000000001"
	createArchivedFixture(t, ctx, store, id)
	assertArchiveOwnership(t, ctx, store, id)
	assertRestorePreservesPin(t, ctx, store, id)
}

func newArchiveStore(t *testing.T) (context.Context, *Store) {
	t.Helper()
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	_, err = db.ExecContext(ctx, `INSERT INTO users (id,name,email,password_hash,created_at)
		VALUES ('u','User','u@example.test','hash',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO projects (id,user_id,name,created_at) VALUES ('p','u','Project',1)`)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, NewStore(db, nil)
}

func createArchivedFixture(t *testing.T, ctx context.Context, store *Store, id string) {
	t.Helper()
	err := store.Create(ctx, domain.Task{
		ID: id, ProjectID: "p", UserID: "u", Title: "Task", Status: domain.StatusDone,
		ProviderID: "provider", Model: "model", CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(1, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func assertArchiveOwnership(t *testing.T, ctx context.Context, store *Store, id string) {
	t.Helper()
	if err := store.Archive(ctx, "other", id, time.Now()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign owner should be hidden: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE tasks SET pinned_at = 7 WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}
	if err := store.Archive(ctx, "u", id, time.Unix(9, 0)); err != nil {
		t.Fatal(err)
	}
}

func assertRestorePreservesPin(t *testing.T, ctx context.Context, store *Store, id string) {
	t.Helper()
	if err := store.Restore(ctx, "u", id); err != nil {
		t.Fatal(err)
	}
	task, err := store.Find(ctx, "u", id)
	if err != nil || task.ArchivedAt != nil || task.PinnedAt == nil || task.PinnedAt.Unix() != 7 {
		t.Fatalf("restore should preserve pin: %+v, %v", task, err)
	}
}
