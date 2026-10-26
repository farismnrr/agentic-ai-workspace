package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
)

type scopes struct {
	err  error
	live bool
	asks int
}

func (s *scopes) TaskLive(context.Context, string, string) (bool, error) {
	s.asks++

	return s.live, s.err
}

func hubWith(t *testing.T, grants bool) *egressproxy.Hub {
	t.Helper()

	base, err := os.MkdirTemp("", "e")
	if err != nil {
		t.Fatal(err)
	}

	hub := egressproxy.NewHub(base, egressproxy.New([]string{"pypi.org"}))
	if grants {
		hub.WithGrants(egressproxy.NewGrants(egressproxy.DefaultLimits(), nil))
	}

	t.Cleanup(func() { hub.Close(); _ = os.RemoveAll(base) })

	return hub
}

func engineFor(hub *egressproxy.Hub, check ScopeCheck) *Engine {
	return NewEngine(EngineConfig{Size: 1, Extras: Extras{Proxy: hub, Scopes: check}})
}

func sockets(t *testing.T, e *Engine, ctx context.Context) (string, bool) {
	t.Helper()

	dir, err := e.socket(ctx, "ws", nil)

	return dir, err == nil
}

func TestWithGrantsOffEveryCommandKeepsTheWorkspaceSocketWhateverItsScope(t *testing.T) {
	hub := hubWith(t, false)
	check := &scopes{live: true}
	e := engineFor(hub, check)

	plain, ok := sockets(t, e, context.Background())
	if !ok {
		t.Fatal("no socket")
	}

	scoped, _ := sockets(t, e, WithTask(context.Background(), "p1", "t1"))
	forged, _ := sockets(t, e, WithTask(context.Background(), "", ""))

	if scoped != plain || forged != plain || check.asks != 0 {
		t.Fatalf("the scope must be ignored when grants are off: %q %q %q (%d checks)", plain, scoped, forged, check.asks)
	}
}

func TestWithGrantsOnACommandWithoutAScopeKeepsTheWorkspaceSocket(t *testing.T) {
	hub := hubWith(t, true)
	e := engineFor(hub, &scopes{live: true})

	dir, ok := sockets(t, e, context.Background())
	want, _ := hub.Dir(context.Background(), "ws")

	if !ok || dir != want {
		t.Fatalf("%q %v, want the workspace socket %q", dir, ok, want)
	}
}

func TestWithGrantsOnALiveTaskGetsItsOwnSocketAndNoOtherTaskShares(t *testing.T) {
	hub := hubWith(t, true)
	e := engineFor(hub, &scopes{live: true})
	r := &Runner{projectID: "p1"}

	a, okA := sockets(t, e, r.bind(WithTask(context.Background(), "p1", "t1")))
	b, okB := sockets(t, e, r.bind(WithTask(context.Background(), "p1", "t2")))
	workspace, _ := hub.Dir(context.Background(), "ws")

	if !okA || !okB || a == b || a == workspace || b == workspace {
		t.Fatalf("%q %q %q", a, b, workspace)
	}

	if _, err := os.Stat(filepath.Join(a, "p.sock")); err != nil {
		t.Fatal(err)
	}
}

var errDown = errors.New("db down")

func TestAScopeThatCannotBeConfirmedGetsNoSocketAtAll(t *testing.T) {
	type scope struct{ project, task string }

	cases := map[string]struct {
		check  ScopeCheck
		runner string
		scopes []scope // applied in order, the last one wins
	}{
		"task not live":  {&scopes{live: false}, "p1", []scope{{"p1", "t1"}}},
		"check fails":    {&scopes{err: errDown}, "p1", []scope{{"p1", "t1"}}},
		"no checker":     {nil, "p1", []scope{{"p1", "t1"}}},
		"other project":  {&scopes{live: true}, "p1", []scope{{"p2", "t1"}}},
		"empty task":     {&scopes{live: true}, "p1", []scope{{"p1", ""}}},
		"empty project":  {&scopes{live: true}, "p1", []scope{{"", "t1"}}},
		"unnamed runner": {&scopes{live: true}, "", []scope{{"p1", "t1"}}},
		"scope replaced": {&scopes{live: true}, "p1", []scope{{"p1", "t1"}, {"p2", "t1"}}},
	}

	for name, c := range cases {
		e := engineFor(hubWith(t, true), c.check)

		ctx := context.Background()
		for _, s := range c.scopes {
			ctx = WithTask(ctx, s.project, s.task)
		}

		if dir, ok := sockets(t, e, (&Runner{projectID: c.runner}).bind(ctx)); ok {
			t.Errorf("%s: got socket %q, want none (and never the workspace's)", name, dir)
		}
	}
}
