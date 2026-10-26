package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	_ "modernc.org/sqlite"
)

func assertWorkQueries(t *testing.T, ctx context.Context, store *Store, workID string) {
	t.Helper()
	if _, found, err := store.FindOp(ctx, workID, "saved"); err != nil || !found {
		t.Fatalf("saved operation missing: found=%v err=%v", found, err)
	}
	assertOpenWorkCounts(t, ctx, store)
	assertStaleWorks(t, ctx, store)
	linked, err := store.ByTask(ctx, "t1")
	if err != nil || linked.ID != workID {
		t.Fatalf("work linked to task: %+v, %v", linked, err)
	}
}

func assertOpenWorkCounts(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	for _, row := range []struct {
		user, project string
		want          int
	}{{"u", "p", 1}, {"other", "", 1}} {
		count, err := store.CountOpen(ctx, row.user, row.project)
		if err != nil || count != row.want {
			t.Fatalf("open works for %s: got=%d want=%d err=%v", row.user, count, row.want, err)
		}
	}
}

func assertStaleWorks(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	works, err := store.StaleOpen(ctx, time.Now().Add(time.Hour))
	if err != nil || len(works) != 2 {
		t.Fatalf("stale open works = %d, %v", len(works), err)
	}
}

func newWorkStore(t *testing.T) (context.Context, *sql.DB, *Store) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), testWorkSchema); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO users(id) VALUES ('u'), ('other')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO projects(id,user_id) VALUES ('p','u'), ('p2','other')`); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db, &workTestIDs{})
	createTaskFixture(t, ctx, store, "t1", "u", "p")
	createTaskFixture(t, ctx, store, "t2", "other", "p2")
	createTaskFixture(t, ctx, store, "t3", "u", "p")
	return ctx, db, store
}

func createTaskFixture(t *testing.T, ctx context.Context, store *Store, id, userID, projectID string) {
	t.Helper()
	err := store.Create(ctx, domain.Task{
		ID: id, ProjectID: projectID, UserID: userID, Title: id,
		Status: domain.StatusDone, ProviderID: "provider", Model: "model", CreatedAt: time.Unix(1, 0),
		UpdatedAt: time.Unix(1, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func createTestWork(t *testing.T, ctx context.Context, store *Store, id, taskID, key string) domain.ExternalWork {
	t.Helper()
	owner, project := "u", "p"
	if taskID == "t2" {
		owner, project = "other", "p2"
	}
	work := domain.ExternalWork{
		ID: id, TaskID: taskID, UserID: owner, ProjectID: project, Client: "client",
		State: domain.WorkOpen, NextStep: 1, IdemKey: key, IdemHash: "request", CreatedAt: time.Unix(1, 0),
		UpdatedAt: time.Unix(1, 0),
	}
	created, err := store.CreateWork(ctx, work)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

type workTestIDs struct{ next int }

func (i *workTestIDs) ID() (string, error) { i.next++; return fmt.Sprintf("event-%d", i.next), nil }
