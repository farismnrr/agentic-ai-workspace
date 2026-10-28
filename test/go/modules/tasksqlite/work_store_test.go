package tasksqlite_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	_ "modernc.org/sqlite"
)

func TestWorkStoreCommitRollsBackAllChangesOnEventFailure(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, db, store, "w1", "t1", "key")
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER fail_work_event BEFORE INSERT ON task_events
		BEGIN SELECT RAISE(ABORT, 'injected event failure'); END`); err != nil {
		t.Fatal(err)
	}
	summary := "must rollback"
	op := application.WorkOp{
		WorkID: work.ID, OpKey: "op1", Tool: "update_plan", ReqHash: "hash",
		Response: `{"ok":true}`, At: time.Now().Unix(),
	}
	_, _, err := store.Commit(ctx, application.WorkChange{
		WorkID: work.ID, UserID: "u", ExpectRev: 0,
		Op: &op, Task: &application.TaskPatch{Summary: &summary}, Events: []domain.Event{{Kind: domain.EventPlan}},
	})
	if err == nil {
		t.Fatal("injected event failure should rollback commit")
	}
	got, err := store.FindWork(ctx, "u", "client", "p", work.ID)
	if err != nil || got.Rev != 0 {
		t.Fatalf("revision escaped rollback: %+v, %v", got, err)
	}
	var gotSummary string
	if queryErr := db.QueryRowContext(ctx, `SELECT summary FROM tasks WHERE id='t1'`).
		Scan(&gotSummary); queryErr != nil || gotSummary != "" {
		t.Fatalf("task patch escaped rollback: %q, %v", gotSummary, queryErr)
	}
	if _, found, opErr := store.FindOp(ctx, work.ID, "op1"); opErr != nil || found {
		t.Fatalf("operation escaped rollback: found=%v err=%v", found, opErr)
	}
}

func TestWorkStoreCreateRollsBackTaskWorkAndRequestWhenRoleEventFails(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER fail_initial_role BEFORE INSERT ON task_events
		WHEN NEW.kind = 'role' BEGIN SELECT RAISE(ABORT, 'injected role failure'); END`); err != nil {
		t.Fatal(err)
	}
	work := domain.ExternalWork{
		ID: "w-role", TaskID: "t-role", UserID: "u", ProjectID: "p", Client: "client",
		State: domain.WorkOpen, RoleKey: "developer", RoleName: "Developer", NextStep: 1,
		CreatedAt: time.Unix(2, 0), UpdatedAt: time.Unix(2, 0),
	}
	task := domain.Task{
		ID: work.TaskID, UserID: work.UserID, ProjectID: work.ProjectID, Source: work.Client,
		WorkID: work.ID, Status: domain.StatusWorking, Title: "request",
		CreatedAt: work.CreatedAt, UpdatedAt: work.UpdatedAt,
	}
	events := []domain.Event{
		{Kind: domain.EventRequest, Text: "request", Time: work.CreatedAt},
		{Kind: domain.EventRole, Text: "Developer", Time: work.CreatedAt},
	}
	if _, err := store.CreateWork(ctx, application.WorkCreate{Task: task, Work: work, Events: events}); err == nil {
		t.Fatal("role event failure should abort creation")
	}
	assertNoCreatedWork(t, ctx, db, work)
}

func assertNoCreatedWork(t *testing.T, ctx context.Context, db *sql.DB, work domain.ExternalWork) {
	t.Helper()
	for table, id := range map[string]string{"tasks": work.TaskID, "external_works": work.ID} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE id = ?`, id).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s row escaped role-event rollback", table)
		}
	}
	var events int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_events WHERE task_id = ?`, work.TaskID).
		Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 0 {
		t.Fatal("initial request event escaped role-event rollback")
	}
}
