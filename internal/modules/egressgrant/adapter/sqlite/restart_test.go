package sqlite_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
)

func TestADatabaseFailureRefusesEverythingAndEmptiesTheRegistry(t *testing.T) {
	w, reg := enforced(t)
	g := w.ask(t, "p1", "t1", "a.example")
	w.approve(t, g, p1own, time.Hour)

	_ = w.db.Close()

	ctx := context.Background()
	if err := w.Verify(ctx, g.ID, "p1", "t1", "a.example"); err == nil || errors.Is(err, application.ErrNotActive) {
		t.Fatalf("verify must fail as a database error: %v", err)
	}

	if live, err := w.TaskLive(ctx, "p1", "t1"); err == nil || live {
		t.Fatalf("task check: %v %v", live, err)
	}

	if err := w.Reconcile(ctx); err == nil || len(reg.IDs()) != 0 {
		t.Fatalf("reconcile %v, held %v: a database that cannot be read must leave nothing enforced", err, reg.IDs())
	}
}

func TestRestartRestoresOnlyLiveApprovedGrants(t *testing.T) {
	w, reg := enforced(t)
	ctx := context.Background()
	live := w.ask(t, "p1", "t1", "live.example")
	other := w.ask(t, "p2", "t3", "other.example")
	ended := w.ask(t, "p1", "t2", "ended.example")
	short := w.ask(t, "p1", "t1", "short.example")
	_ = w.ask(t, "p1", "t1", "pending.example")
	denied := w.ask(t, "p1", "t1", "denied.example")
	revoked := w.ask(t, "p1", "t1", "revoked.example")

	w.approve(t, live, p1own, time.Hour)
	w.approve(t, other, p2own, time.Hour)
	w.approve(t, ended, p1own, time.Hour)
	w.approve(t, short, p1own, time.Minute)
	w.approve(t, revoked, p1own, time.Hour)

	if _, err := w.Deny(ctx, denied.ID, p1own); err != nil {
		t.Fatal(err)
	}

	if _, err := w.Revoke(ctx, revoked.ID, p1own); err != nil {
		t.Fatal(err)
	}

	w.setStatus(t, "t2", "done")
	*w.clock = t0.Add(2 * time.Minute)

	// A restart: the registry is empty and is rebuilt from the database alone.
	fresh := &memRegistry{held: map[string]string{}}
	w.WithRegistry(fresh)

	if err := w.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}

	want := []string{live.ID, other.ID}
	slices.Sort(want)

	if got := fresh.IDs(); !slices.Equal(got, want) {
		t.Fatalf("restored %v, want only %v", got, want)
	}

	_ = reg
}

func TestReconcileDropsAnEntryTheDatabaseDoesNotKnow(t *testing.T) {
	w, reg := enforced(t)
	reg.held["ghost"] = "p1/t1/ghost.example/never"

	if err := w.Reconcile(context.Background()); err != nil || reg.has("ghost") {
		t.Fatalf("%v %v", err, reg.IDs())
	}
}
