package sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
)

func (w world) setStatus(t *testing.T, id, status string) {
	t.Helper()

	query := `UPDATE tasks SET status = ? WHERE id = ?`
	if _, err := w.db.ExecContext(context.Background(), query, status, id); err != nil {
		t.Fatal(err)
	}
}

func (w world) setArchived(t *testing.T, id string, at any) {
	t.Helper()

	query := `UPDATE tasks SET archived_at = ? WHERE id = ?`
	if _, err := w.db.ExecContext(context.Background(), query, at, id); err != nil {
		t.Fatal(err)
	}
}

func TestApprovalIsEnforcedAndRevocationEndsIt(t *testing.T) {
	w, reg := enforced(t)
	g := w.ask(t, "p1", "t1", "a.example")

	if reg.has(g.ID) {
		t.Fatal("a pending request must not be enforced")
	}

	w.approve(t, g, p1own, time.Hour)

	if got := reg.held[g.ID]; got != "p1/t1/a.example/"+t0.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("registry holds %q", got)
	}

	if _, err := w.Revoke(context.Background(), g.ID, p1own); err != nil || reg.has(g.ID) {
		t.Fatalf("revoke: %v, still held: %v", err, reg.has(g.ID))
	}
}

func TestADeniedRequestIsNeverEnforced(t *testing.T) {
	w, reg := enforced(t)
	g := w.ask(t, "p1", "t1", "a.example")

	if _, err := w.Deny(context.Background(), g.ID, p1own); err != nil || len(reg.IDs()) != 0 {
		t.Fatalf("%v %v", err, reg.IDs())
	}
}

func TestIfTheRegistryCannotEnforceTheDecisionFailsAndNothingIsHeld(t *testing.T) {
	w, reg := enforced(t)
	reg.fail = errRegistry
	g := w.ask(t, "p1", "t1", "a.example")

	if _, err := w.Approve(context.Background(), g.ID, p1own, time.Hour); !errors.Is(err, errRegistry) {
		t.Fatalf("got %v", err)
	}

	if len(reg.IDs()) != 0 {
		t.Fatal("nothing may be held")
	}
}

func TestAGrantEndsForGoodWhenItsTaskEnds(t *testing.T) {
	for name, end := range map[string]func(world, *testing.T){
		"done":     func(w world, t *testing.T) { w.setStatus(t, "t1", "done") },
		"failed":   func(w world, t *testing.T) { w.setStatus(t, "t1", "failed") },
		"archived": func(w world, t *testing.T) { w.setArchived(t, "t1", 5) },
	} {
		w, reg := enforced(t)
		g := w.ask(t, "p1", "t1", "a.example")
		w.approve(t, g, p1own, time.Hour)

		end(w, t)

		if err := w.Verify(context.Background(), g.ID, "p1", "t1", "a.example"); !errors.Is(err, application.ErrNotActive) {
			t.Errorf("%s: a new connection must be refused at once: %v", name, err)
		}

		if err := w.Reconcile(context.Background()); err != nil || reg.has(g.ID) {
			t.Fatalf("%s: reconcile %v, still held %v", name, err, reg.has(g.ID))
		}

		// The task is opened again: the grant must not come back.
		w.setStatus(t, "t1", "working")
		w.setArchived(t, "t1", nil)

		err := w.Reconcile(context.Background())
		if err != nil || reg.has(g.ID) || w.count(t, "WHERE status = 'revoked'") != 1 {
			t.Errorf("%s: the grant came back (%v)", name, err)
		}
	}
}

func TestADeletedTaskTakesItsGrantsAndTheRegistryEntryGoes(t *testing.T) {
	w, reg := enforced(t)
	g := w.ask(t, "p1", "t1", "a.example")
	w.approve(t, g, p1own, time.Hour)

	if _, err := w.db.ExecContext(context.Background(), `DELETE FROM tasks WHERE id = 't1'`); err != nil {
		t.Fatal(err)
	}

	if err := w.Reconcile(context.Background()); err != nil || reg.has(g.ID) {
		t.Fatalf("%v %v", err, reg.IDs())
	}

	if err := w.Verify(context.Background(), g.ID, "p1", "t1", "a.example"); err == nil {
		t.Fatal("a deleted task has no grant to verify")
	}
}

func TestVerifyConfirmsEverythingOrRefuses(t *testing.T) {
	w, _ := enforced(t)
	ctx := context.Background()
	g := w.ask(t, "p1", "t1", "a.example")

	if err := w.Verify(ctx, g.ID, "p1", "t1", "a.example"); !errors.Is(err, application.ErrNotActive) {
		t.Fatalf("pending: %v", err)
	}

	w.approve(t, g, p1own, time.Hour)

	if err := w.Verify(ctx, g.ID, "p1", "t1", "A.Example."); err != nil {
		t.Fatalf("a granted connection: %v", err)
	}

	for name, c := range map[string][3]string{
		"other task": {"p1", "t2", "a.example"}, "other project": {"p2", "t1", "a.example"},
		"other host": {"p1", "t1", "b.example"}, "unknown id": {"p1", "t1", "a.example"},
	} {
		id := g.ID
		if name == "unknown id" {
			id = "nope"
		}

		if err := w.Verify(ctx, id, c[0], c[1], c[2]); err == nil {
			t.Errorf("%s was verified", name)
		}
	}

	*w.clock = t0.Add(time.Hour)

	if err := w.Verify(ctx, g.ID, "p1", "t1", "a.example"); !errors.Is(err, application.ErrNotActive) {
		t.Fatalf("expired: %v", err)
	}

	if _, err := w.Revoke(ctx, g.ID, p1own); err == nil {
		t.Log("revoking an expired grant is refused by the domain; fine")
	}
}

func TestARevokedGrantFailsVerifyAtOnce(t *testing.T) {
	w, _ := enforced(t)
	g := w.ask(t, "p1", "t1", "a.example")
	w.approve(t, g, p1own, time.Hour)

	if _, err := w.Revoke(context.Background(), g.ID, p1own); err != nil {
		t.Fatal(err)
	}

	if err := w.Verify(context.Background(), g.ID, "p1", "t1", "a.example"); !errors.Is(err, application.ErrNotActive) {
		t.Fatalf("%v", err)
	}
}
