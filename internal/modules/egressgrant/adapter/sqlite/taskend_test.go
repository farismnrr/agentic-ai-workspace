package sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

func TestTaskEndedClosesEverythingOfThatTaskNow(t *testing.T) {
	w, reg := enforced(t)
	ctx := context.Background()
	live := w.ask(t, "p1", "t1", "live.example")
	waiting := w.ask(t, "p1", "t1", "waiting.example")
	other := w.ask(t, "p1", "t2", "other.example")
	w.approve(t, live, p1own, time.Hour)
	w.approve(t, other, p1own, time.Hour)

	if err := w.TaskEnded(ctx, "t1"); err != nil {
		t.Fatal(err)
	}

	if reg.has(live.ID) || !reg.has(other.ID) {
		t.Fatalf("registry %v: only the ended task's grant may go", reg.IDs())
	}

	ended := w.count(t, "WHERE task_id = 't1' AND status IN ('revoked', 'denied')")
	if ended != 2 || w.count(t, "WHERE status = 'approved'") != 1 {
		t.Fatal("the ended task's rows must be ended for good, the other task's untouched")
	}

	if _, err := w.Approve(ctx, waiting.ID, p1own, 0); !errors.Is(err, domain.ErrNotPending) {
		t.Fatalf("a request of an ended task cannot be approved: %v", err)
	}
}

func TestTaskEndedStillClosesTheRegistryWhenTheTaskRowIsGoneOrTheDatabaseFails(t *testing.T) {
	w, reg := enforced(t)
	ctx := context.Background()
	g := w.ask(t, "p1", "t1", "a.example")
	w.approve(t, g, p1own, time.Hour)

	if _, err := w.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = 't1'`); err != nil {
		t.Fatal(err)
	}

	if err := w.TaskEnded(ctx, "t1"); err != nil || reg.has(g.ID) {
		t.Fatalf("deleted task: %v, held %v", err, reg.has(g.ID))
	}

	g2 := w.ask(t, "p1", "t2", "b.example")
	w.approve(t, g2, p1own, time.Hour)
	_ = w.db.Close()

	if err := w.TaskEnded(ctx, "t2"); err == nil || reg.has(g2.ID) {
		t.Fatalf("a database error must be reported and the registry still cleared: %v, held %v", err, reg.has(g2.ID))
	}
}
