package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

func TestUnauthorizedDecisionsChangeNothing(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	g := w.ask(t, "p1", "t1", "a.example")

	for name, who := range map[string]domain.Decider{
		"other project's owner": p2own, "no rights": {ID: "u1", ProjectID: "p1"}, "no id": {Owner: true, ProjectID: "p1"},
	} {
		if _, err := w.Approve(ctx, g.ID, who, 0); !errors.Is(err, domain.ErrNotAuthorized) {
			t.Errorf("approve, %s: %v", name, err)
		}

		if _, err := w.Deny(ctx, g.ID, who); !errors.Is(err, domain.ErrNotAuthorized) {
			t.Errorf("deny, %s: %v", name, err)
		}
	}

	if w.count(t, "WHERE status = 'pending' AND decided_by = ''") != 1 {
		t.Fatal("the request must still be pending and undecided")
	}
}

func TestAnotherProjectsOwnerCannotRevoke(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p1", "t1", "a.example")
	w.approve(t, g, p1own, 0)

	if _, err := w.Revoke(context.Background(), g.ID, p2own); !errors.Is(err, domain.ErrNotAuthorized) {
		t.Fatalf("got %v", err)
	}

	if got, _ := w.Active(context.Background(), "p1", "t1"); len(got) != 1 {
		t.Fatalf("the grant must still stand: %+v", got)
	}
}

func TestAnAdminMayDecideAcrossProjects(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p2", "t3", "a.example")

	if _, err := w.Approve(context.Background(), g.ID, domain.Decider{ID: "root", Admin: true}, 0); err != nil {
		t.Fatal(err)
	}
}

func TestAnUnknownGrantIsNotFound(t *testing.T) {
	w := newWorld(t)

	if _, err := w.Approve(context.Background(), "nope", p1own, 0); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestDatabaseErrorsAreReportedAsThemselves(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p1", "t1", "a.example")
	ctx := context.Background()

	_ = w.db.Close()

	_, request := w.Request(ctx, "p1", "t1", "b.example", "why", agent)
	_, approve := w.Approve(ctx, g.ID, p1own, 0)
	_, active := w.Active(ctx, "p1", "t1")
	_, list := w.List(ctx, "p1", "t1")

	for name, err := range map[string]error{"request": request, "approve": approve, "active": active, "list": list} {
		if err == nil {
			t.Errorf("%s: a closed database must fail", name)
		}

		for _, masked := range []error{
			application.ErrNotFound, application.ErrConflict, application.ErrScope, application.ErrDuplicate,
		} {
			if errors.Is(err, masked) {
				t.Errorf("%s: a database failure was reported as %v", name, masked)
			}
		}
	}
}

func TestDeletingATaskRemovesItsGrants(t *testing.T) {
	w := newWorld(t)
	w.ask(t, "p1", "t1", "a.example")
	w.ask(t, "p1", "t2", "b.example")

	if _, err := w.db.ExecContext(context.Background(), `DELETE FROM tasks WHERE id = 't1'`); err != nil {
		t.Fatal(err)
	}

	if w.count(t, "") != 1 {
		t.Fatal("only the other task's grant may remain")
	}
}
