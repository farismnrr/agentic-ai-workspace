package bootstrap

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	egressdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
)

var (
	egressAgent = egressdomain.Actor{Kind: egressdomain.ActorAgent, ID: "agent:t1"}
	egressOwner = egressdomain.Decider{ID: "u1", ProjectID: "p1", Owner: true}
)

func TestGrantsAreInertUntilStartedAndTheHubHasNone(t *testing.T) {
	hub := egressHub(t)

	if hub.Enabled() {
		t.Fatal("a hub with no registry must not serve task sockets")
	}

	if _, err := hub.TaskDir(context.Background(), "ws", egressproxy.Scope{Project: "p1", Task: "t1"}); err == nil {
		t.Fatal("task sockets must be refused")
	}
}

func TestApprovalReachesTheProxyAndTaskEndClosesIt(t *testing.T) {
	db, dir := egressDB(t), t.TempDir()
	hub := egressHub(t)
	rt := startEgress(t, db, hub, dir)

	g := approved(t, rt, "t1", "marimo.example.com")
	if !rt.grants.Has(g.ID) {
		t.Fatal("the approval must be enforced")
	}

	scope := egressproxy.Scope{Project: "p1", Task: "t1"}

	lease, err := rt.grants.Lease(scope, "marimo.example.com")
	if err != nil {
		t.Fatal(err)
	}

	held := &sink{}
	lease.Add(held)

	if _, err = db.ExecContext(context.Background(), `UPDATE tasks SET status = 'done' WHERE id = 't1'`); err != nil {
		t.Fatal(err)
	}

	rt.sweep(context.Background())

	if !held.closed || rt.grants.Has(g.ID) {
		t.Fatalf("the task ended: tunnel closed %v, grant held %v", held.closed, rt.grants.Has(g.ID))
	}
}

type sink struct{ closed bool }

func (s *sink) Close() error { s.closed = true; return nil }

func TestRestartRebuildsFromTheDatabaseOnly(t *testing.T) {
	db, dir := egressDB(t), t.TempDir()

	first := startEgress(t, db, egressHub(t), dir)
	live := approved(t, first, "t1", "live.example.com")
	gone := approved(t, first, "t2", "gone.example.com")

	if _, err := db.ExecContext(context.Background(), `UPDATE tasks SET status = 'done' WHERE id = 't2'`); err != nil {
		t.Fatal(err)
	}

	first.stop() // the old process is gone

	second := startEgress(t, db, egressHub(t), dir)

	if !second.grants.Has(live.ID) || second.grants.Has(gone.ID) || len(second.grants.IDs()) != 1 {
		t.Fatalf("restored %v, want only %s", second.grants.IDs(), live.ID)
	}
}

func TestASecondInstanceOnTheSameDataFolderGetsNoGrants(t *testing.T) {
	db, dir := egressDB(t), t.TempDir()
	startEgress(t, db, egressHub(t), dir)

	other := egressHub(t)
	if _, err := startEgressGrants(db, other, dir); err == nil {
		t.Fatal("a second instance must not start grants")
	}

	if other.Enabled() {
		t.Fatal("and its hub must stay without a registry, so it serves no grants at all")
	}
}

func TestADatabaseFailureRefusesConnectionsAndTheSweepEmptiesTheRegistry(t *testing.T) {
	db, dir := egressDB(t), t.TempDir()
	rt := startEgress(t, db, egressHub(t), dir)
	g := approved(t, rt, "t1", "marimo.example.com")

	_ = db.Close()

	scope := egressproxy.Scope{Project: "p1", Task: "t1"}
	verifier := &egressVerifier{service: rt.service}
	if err := verifier.Verify(context.Background(), g.ID, scope, "marimo.example.com"); err == nil {
		t.Fatal("the verifier must fail when the database does")
	}

	if (&egressVerifier{}).Verify(context.Background(), g.ID, scope, "marimo.example.com") == nil {
		t.Fatal("a verifier with no service must refuse")
	}

	rt.sweep(context.Background())

	if len(rt.grants.IDs()) != 0 {
		t.Fatalf("still enforcing %v with an unreadable database", rt.grants.IDs())
	}
}

func TestEachTaskSocketServesOnlyItsOwnTasksGrantsThroughTheRealProxy(t *testing.T) {
	db, dir := egressDB(t), t.TempDir()
	hub := egressHub(t)
	rt := startEgress(t, db, hub, dir)
	approved(t, rt, "t1", "marimo.example.com")

	other, err := hub.TaskDir(context.Background(), "ws", egressproxy.Scope{Project: "p1", Task: "t2"})
	if err != nil {
		t.Fatal(err)
	}

	own, err := hub.TaskDir(context.Background(), "ws", egressproxy.Scope{Project: "p1", Task: "t1"})
	if err != nil || own == other {
		t.Fatalf("%v: every task needs its own folder", err)
	}

	if got := connectVia(t, filepath.Join(other, "p.sock"), "marimo.example.com:443"); got != http.StatusForbidden {
		t.Fatalf("another task's socket: %d", got)
	}
}
