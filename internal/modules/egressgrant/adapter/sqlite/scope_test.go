package sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

func TestApprovalIsKeptAndActiveUntilItsEnd(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	g := w.ask(t, "p1", "t1", "Marimo.Example.")

	if active, err := w.Active(ctx, "p1", "t1"); err != nil || len(active) != 0 {
		t.Fatalf("a pending request is not active: %v %v", active, err)
	}

	w.approve(t, g, p1own, 0)
	*w.clock = t0.Add(time.Hour)

	active, err := w.Active(ctx, "p1", "t1")
	if err != nil || len(active) != 1 || active[0].ID != g.ID || active[0].Host != "marimo.example" {
		t.Fatalf("active: %+v %v", active, err)
	}

	if active[0].DecidedBy != "u1" || !active[0].ExpiresAt.Equal(t0.Add(domain.DefaultTTL)) {
		t.Fatalf("decision not kept: %+v", active[0])
	}
}

func TestRevokeTakesAGrantOutOfActive(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p1", "t1", "a.example")
	w.approve(t, g, p1own, 0)

	if _, err := w.Revoke(context.Background(), g.ID, p1own); err != nil {
		t.Fatal(err)
	}

	if active, _ := w.Active(context.Background(), "p1", "t1"); len(active) != 0 {
		t.Fatalf("a revoked grant is not active: %+v", active)
	}
}

func TestActiveIsOnlyApprovedAndUnexpired(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	w.approve(t, w.ask(t, "p1", "t1", "ok.example"), p1own, time.Hour)
	w.approve(t, w.ask(t, "p1", "t1", "short.example"), p1own, time.Minute)
	_ = w.ask(t, "p1", "t1", "pending.example")

	if _, err := w.Deny(ctx, w.ask(t, "p1", "t1", "denied.example").ID, p1own); err != nil {
		t.Fatal(err)
	}

	if got, err := w.Active(ctx, "p1", "t1"); err != nil || len(got) != 2 {
		t.Fatalf("approved only: %+v %v", got, err)
	}

	*w.clock = t0.Add(time.Minute)

	got, err := w.Active(ctx, "p1", "t1")
	if err != nil || len(got) != 1 || got[0].Host != "ok.example" {
		t.Fatalf("a grant drops out at its end: %+v %v", got, err)
	}

	*w.clock = t0.Add(time.Hour)

	if got, _ = w.Active(ctx, "p1", "t1"); len(got) != 0 {
		t.Fatalf("nothing left: %+v", got)
	}
}

func TestActiveNeverCrossesTaskOrProject(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	w.approve(t, w.ask(t, "p1", "t1", "one.example"), p1own, time.Hour)
	w.approve(t, w.ask(t, "p1", "t2", "two.example"), p1own, time.Hour)
	w.approve(t, w.ask(t, "p2", "t3", "three.example"), p2own, time.Hour)

	for _, wrong := range [][2]string{{"p2", "t1"}, {"p1", "t3"}, {"p2", "t2"}, {"p1", "nope"}, {"", ""}} {
		if got, err := w.Active(ctx, wrong[0], wrong[1]); err != nil || len(got) != 0 {
			t.Errorf("scope %v leaked %+v %v", wrong, got, err)
		}
	}

	if got, _ := w.Active(ctx, "p1", "t2"); len(got) != 1 || got[0].Host != "two.example" {
		t.Fatalf("each task sees its own: %+v", got)
	}
}

func TestARowWithAMismatchedPairNeverCounts(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p1", "t1", "one.example")
	w.approve(t, g, p1own, time.Hour)

	// What a bug or a manual edit could leave behind: the task moves to another project.
	if _, err := w.db.ExecContext(context.Background(), `UPDATE tasks SET project_id = 'p2' WHERE id = 't1'`); err != nil {
		t.Fatal(err)
	}

	for _, project := range []string{"p1", "p2"} {
		if got, err := w.Active(context.Background(), project, "t1"); err != nil || len(got) != 0 {
			t.Errorf("project %s: %+v %v", project, got, err)
		}
	}
}

func TestRequestChecksTheTaskBelongsToTheProject(t *testing.T) {
	w := newWorld(t)

	for _, c := range [][2]string{{"p1", "t3"}, {"p2", "t1"}, {"p1", "missing"}, {"missing", "t1"}} {
		_, err := w.Request(context.Background(), c[0], c[1], "a.example", "why", agent)
		if !errors.Is(err, application.ErrScope) {
			t.Errorf("project %s task %s: %v", c[0], c[1], err)
		}
	}

	if w.count(t, "") != 0 {
		t.Fatal("nothing may be saved")
	}
}

func TestADuplicateLiveRequestIsRefusedUntilItEnds(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	g := w.ask(t, "p1", "t1", "a.example")

	if _, err := w.Request(ctx, "p1", "t1", "A.EXAMPLE.", "again", agent); !errors.Is(err, application.ErrDuplicate) {
		t.Fatalf("pending twice: %v", err)
	}

	_ = w.ask(t, "p1", "t2", "a.example")

	w.approve(t, g, p1own, time.Hour)

	if _, err := w.Request(ctx, "p1", "t1", "a.example", "again", agent); !errors.Is(err, application.ErrDuplicate) {
		t.Fatalf("while approved: %v", err)
	}

	*w.clock = t0.Add(2 * time.Hour)
	_ = w.ask(t, "p1", "t1", "a.example")
}
