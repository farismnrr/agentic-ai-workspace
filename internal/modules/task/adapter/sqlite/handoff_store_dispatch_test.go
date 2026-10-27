package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestHandoffCreatesInternalTaskAndMarksItsOutboxAfterQueueing(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-handoff", "t1", "")
	revision := submitStorePlan(t, ctx, store, work)
	write := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
	if _, err := store.DecidePlan(ctx, write); err != nil {
		t.Fatal(err)
	}
	row, claimed, err := store.ClaimHandoff(ctx, write.HandoffID, time.Unix(20, 0))
	if err != nil || !claimed || row.Revision.QA == "" {
		t.Fatalf("claim handoff = %+v claimed=%v err=%v", row, claimed, err)
	}
	task, event, prompt := handoffTask(work, row, "internal-1", "frozen plan prompt", time.Unix(21, 0))
	linked, created, err := store.CreateHandoffTask(ctx, row, task, event, prompt, time.Unix(21, 0))
	if err != nil || !created || linked.ID != task.ID {
		t.Fatalf("create internal handoff task = %+v created=%v err=%v", linked, created, err)
	}
	assertCreatedHandoffTask(t, ctx, db, store, work, revision, row, linked.ID, prompt)
	if err := store.MarkHandoffDone(ctx, row.ID, time.Unix(22, 0)); err != nil {
		t.Fatal(err)
	}
	assertHandoffRecovery(t, ctx, store, linked.ID)
}

func assertCreatedHandoffTask(t *testing.T, ctx context.Context, db *sql.DB, store *Store,
	work domain.ExternalWork, revision domain.PlanRevision, row application.PlanHandoff, taskID, prompt string,
) {
	t.Helper()
	assertHandoffLink(t, ctx, db, store, work, revision, row, taskID)
	transcript, err := store.LoadTranscript(ctx, taskID)
	if err != nil || len(transcript) != 1 || transcript[0].Blocks[0].Text != prompt {
		t.Fatalf("handoff transcript = %+v, %v", transcript, err)
	}
}

func assertHandoffRecovery(t *testing.T, ctx context.Context, store *Store, taskID string) {
	t.Helper()
	if due, err := store.DueHandoffs(ctx, time.Unix(100, 0), 10); err != nil || len(due) != 0 {
		t.Fatalf("completed handoff remains due: %v, %v", due, err)
	}
	if err := store.ResetHandoffLeases(ctx, time.Unix(23, 0)); err != nil {
		t.Fatal(err)
	}
	if recovering, err := store.RecoveringHandoff(ctx, taskID); err != nil || !recovering {
		t.Fatalf("queued task was not protected for restart recovery: %v, %v", recovering, err)
	}
	if due, err := store.DueHandoffs(ctx, time.Unix(23, 0), 10); err != nil || len(due) != 1 {
		t.Fatalf("restart did not requeue lost scheduler job: %v, %v", due, err)
	}
}

func handoffTask(work domain.ExternalWork, row application.PlanHandoff, id, prompt string, at time.Time) (
	domain.Task, domain.Event, string,
) {
	task := domain.Task{
		ID: id, ProjectID: work.ProjectID, UserID: work.UserID, Title: "Approved outline", Source: "",
		Status: domain.StatusQueued, ProviderID: "provider", Model: "model", CreatedAt: at,
		UpdatedAt: at, OriginTaskID: work.TaskID, PlanFirst: false,
	}
	event := domain.Event{Kind: domain.EventRequest, Text: "Approved plan", Technical: "handoff:" + row.ID, Time: at}
	return task, event, prompt
}

func assertHandoffLink(t *testing.T, ctx context.Context, db *sql.DB, store *Store,
	work domain.ExternalWork, revision domain.PlanRevision, row application.PlanHandoff, taskID string,
) {
	t.Helper()
	assertExternalHandoffState(t, ctx, db, work)
	assertHandoffOutboxLink(t, ctx, db, row.ID, taskID)
	assertHandoffReviewCard(t, ctx, store, work.TaskID, revision.EventID, taskID)
	assertExternalTaskLink(t, ctx, db, store, work, taskID)
}

func assertExternalTaskLink(t *testing.T, ctx context.Context, db *sql.DB, store *Store,
	work domain.ExternalWork, taskID string,
) {
	t.Helper()
	external, err := store.Find(ctx, work.UserID, work.TaskID)
	if err != nil || external.LinkedTaskID != taskID {
		t.Fatalf("external task link = %+v, %v", external, err)
	}
	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, work.TaskID).
		Scan(&status); err != nil || status != string(domain.StatusDone) {
		t.Fatalf("external task status = %s, %v", status, err)
	}
}

func assertExternalHandoffState(t *testing.T, ctx context.Context, db *sql.DB, work domain.ExternalWork) {
	t.Helper()
	var phase, state string
	if workErr := db.QueryRowContext(ctx, `SELECT plan_state,state FROM external_works WHERE id = ?`, work.ID).
		Scan(&phase, &state); workErr != nil || phase != string(domain.PlanHandedOff) ||
		state != string(domain.WorkCompleted) {
		t.Fatalf("handed-off work = %s/%s, %v", phase, state, workErr)
	}
}

func assertHandoffOutboxLink(t *testing.T, ctx context.Context, db *sql.DB, handoffID, taskID string) {
	t.Helper()
	var outboxTask string
	if err := db.QueryRowContext(ctx, `SELECT internal_task_id FROM external_plan_handoffs WHERE id = ?`, handoffID).
		Scan(&outboxTask); err != nil || outboxTask != taskID {
		t.Fatalf("outbox task = %s, %v", outboxTask, err)
	}
}

func assertHandoffReviewCard(t *testing.T, ctx context.Context, store *Store,
	taskID, eventID, internalID string,
) {
	t.Helper()
	card, err := store.FindEvent(ctx, taskID, eventID)
	if err != nil || card.PlanReview.HandoffState != "handed_off" || card.PlanReview.TaskID != internalID {
		t.Fatalf("handed-off card = %+v, %v", card.PlanReview, err)
	}
}
