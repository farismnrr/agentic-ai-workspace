package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	_ "modernc.org/sqlite"
)

func TestWorkStoreCommitUsesRevisionCAS(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w1", "t1", "key")
	event := domain.Event{Kind: domain.EventRole, Text: "Developer"}
	op := application.WorkOp{
		WorkID: work.ID, OpKey: "saved", Tool: "set_role", ReqHash: "hash",
		Response: `{"ok":true}`, At: time.Now().Unix(),
	}
	events, rev, err := store.Commit(ctx, application.WorkChange{
		WorkID: work.ID, UserID: "u", ExpectRev: 0, Op: &op, Events: []domain.Event{event},
	})
	if err != nil || rev != 1 || len(events) != 1 {
		t.Fatalf("commit: events=%d rev=%d err=%v", len(events), rev, err)
	}
	staleChange := application.WorkChange{WorkID: work.ID, UserID: "u", ExpectRev: 0}
	if _, _, staleErr := store.Commit(ctx, staleChange); !errors.Is(staleErr, domain.ErrStaleRevision) {
		t.Fatalf("stale revision error = %v", staleErr)
	}
}

func TestWorkStoreCommitRollsBackAllChangesOnEventFailure(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w1", "t1", "key")
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

func TestWorkStoreIdempotencyAndQueries(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	first := createTestWork(t, ctx, store, "w1", "t1", "same-key")
	assertAtomicWorkCreate(t, ctx, store, first)
	duplicate := createTestWork(t, ctx, store, "w2", "t3", "same-key")
	if duplicate.ID != first.ID {
		t.Fatalf("duplicate key returned work %q, want %q", duplicate.ID, first.ID)
	}
	if _, err := store.Find(ctx, "u", "t3"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("idempotent replay left a duplicate task: %v", err)
	}
	createTestWork(t, ctx, store, "w3", "t2", "")
	op := application.WorkOp{
		WorkID: first.ID, OpKey: "saved", Tool: "set_role", ReqHash: "hash",
		Response: `{"ok":true}`, At: time.Now().Unix(),
	}
	if _, _, err := store.Commit(ctx, application.WorkChange{WorkID: first.ID, UserID: "u", Op: &op}); err != nil {
		t.Fatal(err)
	}
	assertWorkQueries(t, ctx, store, first.ID)
}

func assertAtomicWorkCreate(t *testing.T, ctx context.Context, store *Store, work domain.ExternalWork) {
	t.Helper()
	task, err := store.Find(ctx, "u", work.TaskID)
	if err != nil || task.WorkID != work.ID || task.Status != domain.StatusWorking || task.Source != "client" {
		t.Fatalf("created task link: %+v, %v", task, err)
	}
	events, err := store.EventsAfter(ctx, work.TaskID, 0)
	if err != nil || len(events) != 1 || events[0].Kind != domain.EventRequest {
		t.Fatalf("initial request event: %+v, %v", events, err)
	}
}

func TestWorkStoreOwnerIsolationAndCascade(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	first := createTestWork(t, ctx, store, "w1", "t1", "")
	op := application.WorkOp{
		WorkID: first.ID, OpKey: "saved", Tool: "set_role", ReqHash: "hash",
		Response: `{"ok":true}`, At: time.Now().Unix(),
	}
	if _, _, err := store.Commit(ctx, application.WorkChange{WorkID: first.ID, UserID: "u", Op: &op}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FindWork(ctx, "other", "client", "p2", first.ID); !errors.Is(err, domain.ErrUnknownWork) {
		t.Fatalf("wrong owner must not see work: %v", err)
	}
	change := application.WorkChange{WorkID: first.ID, UserID: "other", ExpectRev: 0}
	if _, _, ownerErr := store.Commit(ctx, change); !errors.Is(ownerErr, domain.ErrUnknownWork) {
		t.Fatalf("wrong owner commit error = %v", ownerErr)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM tasks WHERE id='t1'`); err != nil {
		t.Fatal(err)
	}
	var works, ops int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_works WHERE id=?`, first.ID).
		Scan(&works); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM external_work_ops WHERE work_id=?`, first.ID).
		Scan(&ops); err != nil {
		t.Fatal(err)
	}
	if works != 0 || ops != 0 {
		t.Fatalf("task delete did not cascade: works=%d ops=%d", works, ops)
	}
}
