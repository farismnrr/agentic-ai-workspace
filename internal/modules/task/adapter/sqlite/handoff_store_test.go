package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestBusyLinkedChatLeavesHandoffPendingAndFollowUpIsIdempotent(t *testing.T) {
	for _, status := range []domain.Status{domain.StatusWorking, domain.StatusDone} {
		t.Run(string(status), func(t *testing.T) {
			ctx, db, store, _, row, linked, followup := prepareFollowUp(t, status)
			if status == domain.StatusWorking {
				assertBusyFollowUp(t, ctx, db, store, row, linked, followup)
				return
			}
			assertIdempotentFollowUp(t, ctx, db, store, row, linked, followup)
		})
	}
}

func prepareFollowUp(t *testing.T, status domain.Status) (
	context.Context, *sql.DB, *Store, domain.ExternalWork, application.PlanHandoff, domain.Task, domain.Event,
) {
	t.Helper()
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-follow-"+string(status), "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	write := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
	if _, err := store.DecidePlan(ctx, write); err != nil {
		t.Fatal(err)
	}
	row, _, err := store.ClaimHandoff(ctx, write.HandoffID, time.Unix(20, 0))
	if err != nil {
		t.Fatal(err)
	}
	linked, _, err := store.CreateLinkedTask(ctx, work.TaskID, linkedTaskFixture(90),
		domain.Event{Kind: domain.EventRequest, Text: "Owner conversation"}, "owner prompt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE id = ?`, status, linked.ID); err != nil {
		t.Fatal(err)
	}
	followup := domain.Event{
		Kind: domain.EventRequest, Text: "Approved plan", Technical: "handoff:" + row.ID,
		Time: time.Unix(21, 0),
	}
	return ctx, db, store, work, row, linked, followup
}

func assertBusyFollowUp(t *testing.T, ctx context.Context, db *sql.DB, store *Store,
	row application.PlanHandoff, linked domain.Task, followup domain.Event,
) {
	t.Helper()
	err := store.FollowHandoff(ctx, row, linked, followup, "snapshot", time.Unix(21, 0))
	if !errors.Is(err, domain.ErrBusy) {
		t.Fatalf("busy linked chat error = %v", err)
	}
	var state string
	_ = db.QueryRowContext(ctx, `SELECT state FROM external_plan_handoffs WHERE id = ?`, row.ID).Scan(&state)
	if state != "pending" {
		t.Fatalf("busy handoff state = %s", state)
	}
}

func assertIdempotentFollowUp(t *testing.T, ctx context.Context, db *sql.DB, store *Store,
	row application.PlanHandoff, linked domain.Task, followup domain.Event,
) {
	t.Helper()
	for _, at := range []time.Time{time.Unix(21, 0), time.Unix(22, 0)} {
		if err := store.FollowHandoff(ctx, row, linked, followup, "snapshot", at); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_events WHERE task_id = ?`, linked.ID).Scan(&count)
	if count != 2 {
		t.Fatalf("retry added duplicate handoff event: %d", count)
	}
}

func TestRecoveryResetsALeaseOnAPendingApproval(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-pending-recovery", "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	write := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
	if _, err := store.DecidePlan(ctx, write); err != nil {
		t.Fatal(err)
	}
	if _, claimed, err := store.ClaimHandoff(ctx, write.HandoffID, time.Unix(20, 0)); err != nil || !claimed {
		t.Fatalf("claim pending handoff = %v, %v", claimed, err)
	}
	if due, err := store.DueHandoffs(ctx, time.Unix(20, 0), 10); err != nil || len(due) != 0 {
		t.Fatalf("unexpired lease was listed: %v, %v", due, err)
	}
	if err := store.ResetHandoffLeases(ctx, time.Unix(20, 0)); err != nil {
		t.Fatal(err)
	}
	if due, err := store.DueHandoffs(ctx, time.Unix(20, 0), 10); err != nil || len(due) != 1 || due[0] != write.HandoffID {
		t.Fatalf("pending handoff was not recovered: %v, %v", due, err)
	}
}

func TestFailedHandoffReachesEightAttemptsAndApprovalRearmsIt(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-rearm", "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	write := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
	if _, err := store.DecidePlan(ctx, write); err != nil {
		t.Fatal(err)
	}
	failHandoffAttempts(t, ctx, store, write.HandoffID)
	assertFailedHandoff(t, ctx, db, write.HandoffID)
	if _, err := store.DecidePlan(ctx, write); err != nil {
		t.Fatalf("repeat approval did not rearm the failed handoff: %v", err)
	}
	assertRearmedHandoff(t, ctx, db, write.HandoffID)
}

func failHandoffAttempts(t *testing.T, ctx context.Context, store *Store, handoffID string) {
	t.Helper()
	for attempt := range 8 {
		now := time.Unix(int64(10_000+attempt*10_000), 0)
		if _, claimed, err := store.ClaimHandoff(ctx, handoffID, now); err != nil || !claimed {
			t.Fatalf("claim attempt %d = %v, %v", attempt+1, claimed, err)
		}
		if err := store.FailHandoff(ctx, handoffID, "dispatch_failed", now); err != nil {
			t.Fatal(err)
		}
	}
}

func assertFailedHandoff(t *testing.T, ctx context.Context, db *sql.DB, handoffID string) {
	t.Helper()
	var state, category string
	if err := db.QueryRowContext(ctx, `SELECT state,last_error FROM external_plan_handoffs WHERE id = ?`,
		handoffID).Scan(&state, &category); err != nil || state != "failed" || category != "dispatch_failed" {
		t.Fatalf("exhausted handoff = %s/%s, %v", state, category, err)
	}
}

func assertRearmedHandoff(t *testing.T, ctx context.Context, db *sql.DB, handoffID string) {
	t.Helper()
	var state, category string
	var attempts int
	if err := db.QueryRowContext(ctx, `SELECT state,attempts,last_error FROM external_plan_handoffs WHERE id = ?`,
		handoffID).Scan(&state, &attempts, &category); err != nil || state != "pending" ||
		attempts != 0 || category != "" {
		t.Fatalf("rearmed handoff = %s/%d/%s, %v", state, attempts, category, err)
	}
}
