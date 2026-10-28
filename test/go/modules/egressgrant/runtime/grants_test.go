package runtime_test

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	egressruntime "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/runtime"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
	egressapptest "github.com/farismnrr/agentic-ai-workspace/test/go/apptest/egress"
)

func TestGrantsAreInertUntilStartedAndTheHubHasNone(t *testing.T) {
	hub := egressapptest.Hub(t)

	if hub.Enabled() {
		t.Fatal("a hub with no registry must not serve task sockets")
	}

	if _, err := hub.TaskDir(context.Background(), "ws", egressproxy.Scope{Project: "p1", Task: "t1"}); err == nil {
		t.Fatal("task sockets must be refused")
	}
}

func TestApprovalReachesTheProxyAndTaskEndClosesIt(t *testing.T) {
	db, dir := egressapptest.DB(t), t.TempDir()
	hub := egressapptest.Hub(t)
	rt := egressapptest.Start(t, db, hub, dir)

	g := egressapptest.Approve(t, rt, "t1", "marimo.example.com")
	if !rt.Registry().Has(g.ID) {
		t.Fatal("the approval must be enforced")
	}

	scope := egressproxy.Scope{Project: "p1", Task: "t1"}

	lease, err := rt.Registry().Lease(scope, "marimo.example.com")
	if err != nil {
		t.Fatal(err)
	}

	held := &sink{}
	lease.Add(held)

	if _, err = db.ExecContext(context.Background(), `UPDATE tasks SET status = 'done' WHERE id = 't1'`); err != nil {
		t.Fatal(err)
	}

	rt.Sweep(context.Background())

	if !held.closed || rt.Registry().Has(g.ID) {
		t.Fatalf("the task ended: tunnel closed %v, grant held %v", held.closed, rt.Registry().Has(g.ID))
	}
}

type sink struct{ closed bool }

func (s *sink) Close() error { s.closed = true; return nil }

func TestRestartRebuildsFromTheDatabaseOnly(t *testing.T) {
	db, dir := egressapptest.DB(t), t.TempDir()

	first := egressapptest.Start(t, db, egressapptest.Hub(t), dir)
	live := egressapptest.Approve(t, first, "t1", "live.example.com")
	gone := egressapptest.Approve(t, first, "t2", "gone.example.com")

	if _, err := db.ExecContext(context.Background(), `UPDATE tasks SET status = 'done' WHERE id = 't2'`); err != nil {
		t.Fatal(err)
	}

	first.Close() // the old process is gone

	second := egressapptest.Start(t, db, egressapptest.Hub(t), dir)

	if !second.Registry().Has(live.ID) || second.Registry().Has(gone.ID) || len(second.Registry().IDs()) != 1 {
		t.Fatalf("restored %v, want only %s", second.Registry().IDs(), live.ID)
	}
}

func TestASecondInstanceOnTheSameDataFolderGetsNoGrants(t *testing.T) {
	db, dir := egressapptest.DB(t), t.TempDir()
	egressapptest.Start(t, db, egressapptest.Hub(t), dir)

	other := egressapptest.Hub(t)
	if _, err := egressruntime.Start(db, other, dir); err == nil {
		t.Fatal("a second instance must not start grants")
	}

	if other.Enabled() {
		t.Fatal("and its hub must stay without a registry, so it serves no grants at all")
	}
}

func TestADatabaseFailureRefusesConnectionsAndTheSweepEmptiesTheRegistry(t *testing.T) {
	db, dir := egressapptest.DB(t), t.TempDir()
	hub := egressapptest.Hub(t)
	rt := egressapptest.Start(t, db, hub, dir)
	g := egressapptest.Approve(t, rt, "t1", "marimo.example.com")
	if !rt.Registry().Has(g.ID) {
		t.Fatal("the grant should be live before database failure")
	}
	scope := egressproxy.Scope{Project: "p1", Task: "t1"}
	taskDir, err := hub.TaskDir(context.Background(), "ws", scope)
	if err != nil {
		t.Fatal(err)
	}

	_ = db.Close()

	if got := egressapptest.ConnectVia(t, filepath.Join(taskDir, "p.sock"),
		"marimo.example.com:443"); got != http.StatusForbidden {
		t.Fatalf("an unreadable grant database must fail closed, got %d", got)
	}

	rt.Sweep(context.Background())

	if len(rt.Registry().IDs()) != 0 {
		t.Fatalf("still enforcing %v with an unreadable database", rt.Registry().IDs())
	}
}

func TestEachTaskSocketServesOnlyItsOwnTasksGrantsThroughTheRealProxy(t *testing.T) {
	db, dir := egressapptest.DB(t), t.TempDir()
	hub := egressapptest.Hub(t)
	rt := egressapptest.Start(t, db, hub, dir)
	egressapptest.Approve(t, rt, "t1", "marimo.example.com")

	other, err := hub.TaskDir(context.Background(), "ws", egressproxy.Scope{Project: "p1", Task: "t2"})
	if err != nil {
		t.Fatal(err)
	}

	own, err := hub.TaskDir(context.Background(), "ws", egressproxy.Scope{Project: "p1", Task: "t1"})
	if err != nil || own == other {
		t.Fatalf("%v: every task needs its own folder", err)
	}

	if got := egressapptest.ConnectVia(t, filepath.Join(other, "p.sock"),
		"marimo.example.com:443"); got != http.StatusForbidden {
		t.Fatalf("another task's socket: %d", got)
	}
}
