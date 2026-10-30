package tasksqlite_test

import (
	"context"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"testing"
	"time"
)

func TestStoppedTaskCannotBeReopenedByLateWorker(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	created := createTestWork(t, ctx, db, store, "manual-stop-w", "manual-stop-t", "manual-stop-key")
	now := time.Now()
	if err := store.SetStatus(ctx, created.TaskID, domain.StatusDone, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SetStatus(ctx, created.TaskID, domain.StatusPlanning, now); err == nil {
		t.Fatal("late worker reopened completed task as planning")
	}
	if err := store.SetStatus(ctx, created.TaskID, domain.StatusWorking, now); err == nil {
		t.Fatal("late worker reopened completed task as working")
	}
	if err := store.Finish(ctx, created.TaskID, "unexpected review", nil, now); err == nil {
		t.Fatal("late worker reopened completed task as review")
	}
	changed, err := store.FailUnlessDone(ctx, created.TaskID, now)
	if err != nil || changed {
		t.Fatalf("late failure overwrote done: %v %v", changed, err)
	}
	var status string
	if err := db.QueryRowContext(context.Background(), `SELECT status FROM tasks WHERE id=?`, created.TaskID).Scan(&status); err != nil || status != "done" {
		t.Fatalf("status reverted to %q: %v", status, err)
	}
}
