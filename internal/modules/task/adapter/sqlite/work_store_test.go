package sqlite

import (
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
	got, err := store.FindWork(ctx, "u", work.ID)
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

func TestWorkStoreIdempotencyAndQueries(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	first := createTestWork(t, ctx, store, "w1", "t1", "same-key")
	duplicate := createTestWork(t, ctx, store, "w2", "t3", "same-key")
	if duplicate.ID != first.ID {
		t.Fatalf("duplicate key returned work %q, want %q", duplicate.ID, first.ID)
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
	if _, err := store.FindWork(ctx, "other", first.ID); !errors.Is(err, domain.ErrUnknownWork) {
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
