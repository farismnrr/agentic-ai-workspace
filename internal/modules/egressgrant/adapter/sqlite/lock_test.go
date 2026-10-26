package sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

// This pins the behaviour the write-lock strategy relies on in the SQLite driver we ship (modernc, WAL, deferred
// transactions): a decision whose first statement is a write waits for the lock, then reads what the holder
// committed. It does not fail with a stale snapshot, which is what a read-then-write transaction does. A
// connection-wide `_txlock=immediate` would give the same wait, but for every transaction of every module, so the
// write statement is used instead, only where a decision needs it.
func TestADecisionWaitsForTheWriteLockAndThenSeesWhatWasCommitted(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	g := w.ask(t, "p1", "t1", "a.example")

	holder, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = holder.Rollback() }()

	if _, err = holder.ExecContext(ctx, `UPDATE egress_grants SET status = 'denied' WHERE id = ?`, g.ID); err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)

	go func() {
		_, e := w.Approve(ctx, g.ID, p1own, time.Hour)
		result <- only(e)
	}()

	select {
	case e := <-result:
		t.Fatalf("the decision did not wait for the lock holder: %v", e)
	case <-time.After(300 * time.Millisecond):
	}

	if err = holder.Commit(); err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-result:
		if !errors.Is(e, domain.ErrNotPending) {
			t.Fatalf("after the lock holder denied it, approving must be refused by the domain rule, got %v", e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the decision never finished")
	}
}
