package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestCreateLinkedTaskRacesReuseOneInternalConversation(t *testing.T) {
	ctx, db, store := newConcurrentWorkStore(t)
	if _, err := db.ExecContext(ctx, `UPDATE tasks SET source='ChatGPT' WHERE id='t1'`); err != nil {
		t.Fatal(err)
	}
	chosen := concurrentLinkedTaskID(t, ctx, store)
	assertSingleLinkedConversation(t, ctx, db, chosen)
	listed, err := store.List(ctx, "u", "p", 100)
	if err != nil {
		t.Fatal(err)
	}
	assertLinkedTaskPayloadSource(t, listed, chosen)
	if _, created, err := store.CreateLinkedTask(ctx, "t1", linkedTaskFixture(100),
		domain.Event{Kind: domain.EventRequest, Text: "retry"}, "retry context"); err != nil || created {
		t.Fatalf("retry should reuse prior linked task: created=%v err=%v", created, err)
	}
}

func TestCreateLinkedTaskReplacesArchivedConversation(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	if _, err := db.ExecContext(ctx, `UPDATE tasks SET source='ChatGPT' WHERE id='t1'`); err != nil {
		t.Fatal(err)
	}
	first, created, err := store.CreateLinkedTask(ctx, "t1", linkedTaskFixture(100),
		domain.Event{Kind: domain.EventRequest, Text: "first"}, "first context")
	if err != nil || !created {
		t.Fatalf("create first linked task: created=%v err=%v", created, err)
	}
	_, err = db.ExecContext(ctx, `UPDATE tasks SET archived_at=10 WHERE id=?`, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, created, err := store.CreateLinkedTask(ctx, "t1", linkedTaskFixture(101),
		domain.Event{Kind: domain.EventRequest, Text: "second"}, "second context")
	if err != nil || !created || second.ID == first.ID {
		t.Fatalf("create replacement linked task: id=%q created=%v err=%v", second.ID, created, err)
	}
	var linkedID string
	if err := db.QueryRowContext(ctx, `SELECT linked_task_id FROM tasks WHERE id='t1'`).Scan(&linkedID); err != nil {
		t.Fatal(err)
	}
	if linkedID != second.ID {
		t.Fatalf("external task link = %q, want %q", linkedID, second.ID)
	}
}

func concurrentLinkedTaskID(t *testing.T, ctx context.Context, store *Store) string {
	t.Helper()
	const sends = 12
	var wait sync.WaitGroup
	results := make(chan string, sends)
	errors := make(chan error, sends)
	for i := 0; i < sends; i++ {
		wait.Add(1)
		go func(n int) {
			defer wait.Done()
			task := linkedTaskFixture(n)
			got, _, err := store.CreateLinkedTask(ctx, "t1", task,
				domain.Event{Kind: domain.EventRequest, Text: fmt.Sprintf("message %d", n), Time: time.Now()},
				fmt.Sprintf("context %d", n))
			if err != nil {
				errors <- err
				return
			}
			results <- got.ID
		}(i)
	}
	wait.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Errorf("concurrent create: %v", err)
	}
	var chosen string
	for id := range results {
		if chosen == "" {
			chosen = id
		}
		if id != chosen {
			t.Fatalf("concurrent sends returned different linked tasks %q and %q", chosen, id)
		}
	}
	return chosen
}

func assertSingleLinkedConversation(t *testing.T, ctx context.Context, db *sql.DB, chosen string) {
	t.Helper()
	var linked, internal, events, blocks int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE id='t1' AND linked_task_id=?`, chosen).
		Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE origin_task_id='t1' AND source='' AND work_id=''`).
		Scan(&internal); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM task_events WHERE task_id=?`, chosen).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM task_transcript WHERE task_id=?`, chosen).Scan(&blocks); err != nil {
		t.Fatal(err)
	}
	if linked != 1 || internal != 1 || events != 1 || blocks != 1 {
		t.Fatalf("created link/task/event/transcript = %d/%d/%d/%d", linked, internal, events, blocks)
	}
}

func TestCreateLinkedTaskRejectsWrongOwnerProjectAndArchivedExternalTask(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup string
		id    string
		user  string
		proj  string
	}{
		{name: "wrong owner", id: "t2", user: "u", proj: "p"},
		{name: "wrong project", setup: `INSERT INTO projects(id,user_id) VALUES ('p3','u')`, id: "t1", user: "u", proj: "p3"},
		{name: "archived", setup: `UPDATE tasks SET archived_at=10 WHERE id='t1'`, id: "t1", user: "u", proj: "p"},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertInvalidLinkedTask(t, test.id, test.user, test.proj, test.setup)
		})
	}
}

func assertInvalidLinkedTask(t *testing.T, id, user, project, setup string) {
	t.Helper()
	ctx, db, store := newWorkStore(t)
	if _, err := db.ExecContext(ctx, `UPDATE tasks SET source='ChatGPT' WHERE id='t1'`); err != nil {
		t.Fatal(err)
	}
	if setup != "" {
		if _, err := db.ExecContext(ctx, setup); err != nil {
			t.Fatal(err)
		}
	}
	task := linkedTaskFixture(0)
	task.UserID, task.ProjectID = user, project
	if _, _, err := store.CreateLinkedTask(ctx, id, task,
		domain.Event{Kind: domain.EventRequest}, "prompt"); err == nil {
		t.Fatal("link creation should reject invalid external task")
	}
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tasks WHERE origin_task_id=?`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("invalid request left %d internal tasks", count)
	}
}

func linkedTaskFixture(n int) domain.Task {
	now := time.Now().UTC()
	return domain.Task{
		ID: fmt.Sprintf("linked-%d", n), ProjectID: "p", UserID: "u", Title: "Help",
		Status: domain.StatusQueued, CreatedAt: now, UpdatedAt: now, Plan: []domain.PlanStep{}, OriginTaskID: "t1",
	}
}
