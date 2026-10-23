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
