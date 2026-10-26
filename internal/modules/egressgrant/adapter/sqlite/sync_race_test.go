package sqlite_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Approve, revoke and the sweep run against the same grants at once. Whatever order they land in, once everything
// has finished the registry must say exactly what the database says: held if and only if approved and unexpired.
func TestApprovalRevocationAndSweepsRacingLeaveTheRegistryEqualToTheDatabase(t *testing.T) {
	w, reg := enforced(t)
	ctx := context.Background()

	const rounds = 12

	for round := range rounds {
		g := w.ask(t, "p1", "t1", "r"+strconv.Itoa(round)+".example")

		raceDecisions(w, g.ID)

		rows, err := w.List(ctx, "p1", "t1")
		if err != nil {
			t.Fatal(err)
		}

		for _, row := range rows {
			if row.ID == g.ID && reg.has(g.ID) != row.Active(*w.clock) {
				t.Fatalf("round %d: database says %s, registry holds it: %v", round, row.Status, reg.has(g.ID))
			}
		}
	}
}

// raceDecisions runs two approvals, two revokes and two sweeps for one grant at the same moment.
func raceDecisions(w world, id string) {
	ctx := context.Background()

	var wg sync.WaitGroup

	start := make(chan struct{})

	for i := range 6 {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			switch i % 3 {
			case 0:
				_, _ = w.Approve(ctx, id, p1own, time.Hour)
			case 1:
				_, _ = w.Revoke(ctx, id, p1own)
			default:
				_ = w.Reconcile(ctx)
			}
		}()
	}

	close(start)
	wg.Wait()
}

func TestConcurrentSweepsAndTaskEndNeverResurrectAGrant(t *testing.T) {
	w, reg := enforced(t)
	ctx := context.Background()
	g := w.ask(t, "p1", "t1", "a.example")
	w.approve(t, g, p1own, time.Hour)

	var wg sync.WaitGroup

	start := make(chan struct{})

	for i := range 12 {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			if i == 0 {
				_, _ = w.db.ExecContext(ctx, `UPDATE tasks SET status = 'done' WHERE id = 't1'`)
			}

			_ = w.Reconcile(ctx)
		}()
	}

	close(start)
	wg.Wait()

	if err := w.Reconcile(ctx); err != nil || reg.has(g.ID) {
		t.Fatalf("%v held=%v", err, reg.has(g.ID))
	}
}
