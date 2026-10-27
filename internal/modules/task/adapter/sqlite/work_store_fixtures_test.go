package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	_ "modernc.org/sqlite"
)

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
	if _, err := store.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, taskID); err != nil {
		t.Fatal(err)
	}
	task := domain.Task{
		ID: taskID, ProjectID: project, UserID: owner, Title: taskID, Source: "client",
		Status: domain.StatusWorking, ProviderID: "", Model: "", WorkID: id,
		CreatedAt: work.CreatedAt, UpdatedAt: work.UpdatedAt,
	}
	event := domain.Event{Kind: domain.EventRequest, Text: "request", Time: work.CreatedAt}
	created, err := store.CreateWork(ctx, application.WorkCreate{Task: task, Work: work, Events: []domain.Event{event}})
	if err != nil {
		t.Fatal(err)
	}
	return created
}

type workTestIDs struct{ next int }

func (i *workTestIDs) ID() (string, error) { i.next++; return fmt.Sprintf("event-%d", i.next), nil }
