package runtime_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	egressruntime "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/runtime"
	tasksqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/sqlite"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
	egressapptest "github.com/farismnrr/agentic-ai-workspace/test/go/apptest/egress"
)

var errStore = errors.New("store failed")

type endLog struct {
	ids []string
	mu  sync.Mutex
}

func (l *endLog) add(_ context.Context, id string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.ids = append(l.ids, id)
}

func (l *endLog) got() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := slices.Clone(l.ids)
	slices.Sort(out)

	return out
}

type fakeStore struct {
	taskapp.TaskStore
	err   error
	tasks []taskdomain.Task
}

func (f fakeStore) SetStatus(context.Context, string, taskdomain.Status, time.Time) error {
	return f.err
}
func (f fakeStore) Archive(context.Context, string, string, time.Time) error { return f.err }
func (f fakeStore) DeleteProject(context.Context, string, string) error      { return f.err }
func (f fakeStore) DeleteAll(context.Context, string) error                  { return f.err }

func (f fakeStore) DeleteOwned(_ context.Context, _ string, ids []string) ([]string, error) {
	return ids, f.err
}

func (f fakeStore) All(context.Context, string, string, int) ([]taskdomain.Task, error) {
	return f.tasks, nil
}

func TestEveryWayATaskEndsIsReportedAndNothingElseIs(t *testing.T) {
	ctx := context.Background()
	end := &endLog{}
	store := egressruntime.Tasks(fakeStore{tasks: []taskdomain.Task{{ID: "a"}, {ID: "b"}}}, end.add)

	for _, status := range []taskdomain.Status{
		taskdomain.StatusWorking, taskdomain.StatusPlanning, taskdomain.StatusQueued, taskdomain.StatusReview,
		taskdomain.StatusNeedsInput,
	} {
		_ = store.SetStatus(ctx, "open", status, time.Now())
	}

	if len(end.got()) != 0 {
		t.Fatalf("a task that is still open was reported: %v", end.got())
	}

	_ = store.SetStatus(ctx, "done", taskdomain.StatusDone, time.Now())
	_ = store.SetStatus(ctx, "failed", taskdomain.StatusFailed, time.Now())
	_ = store.Archive(ctx, "u1", "archived", time.Now())
	_, _ = store.DeleteOwned(ctx, "u1", []string{"d1", "d2"})
	_ = store.DeleteProject(ctx, "u1", "p1")
	_ = store.DeleteAll(ctx, "u1")

	want := []string{"a", "a", "archived", "b", "b", "d1", "d2", "done", "failed"}
	if got := end.got(); !slices.Equal(got, want) {
		t.Fatalf("reported %v, want %v", got, want)
	}
}

func TestAFailedStoreCallEndsNothing(t *testing.T) {
	ctx := context.Background()
	end := &endLog{}
	store := egressruntime.Tasks(fakeStore{err: errStore, tasks: []taskdomain.Task{{ID: "a"}}}, end.add)

	if store.SetStatus(ctx, "x", taskdomain.StatusDone, time.Now()) == nil ||
		store.Archive(ctx, "u", "y", time.Now()) == nil || store.DeleteProject(ctx, "u", "p") == nil ||
		store.DeleteAll(ctx, "u") == nil {
		t.Fatal("errors must pass through")
	}

	if _, err := store.DeleteOwned(ctx, "u", []string{"z"}); err == nil {
		t.Fatal("errors must pass through")
	}
}

// openTunnel approves a host for a task and opens a tunnel under it.
func openTunnel(t *testing.T, rt *egressruntime.Runtime, task string) (*sink, string) {
	t.Helper()

	g := egressapptest.Approve(t, rt, task, "marimo.example.com")

	lease, err := rt.Registry().Lease(egressproxy.Scope{Project: "p1", Task: task}, "marimo.example.com")
	if err != nil {
		t.Fatal(err)
	}

	held := &sink{}
	lease.Add(held)

	return held, g.ID
}

func TestEndingATaskClosesItsTunnelsAtOnceWithoutASweep(t *testing.T) {
	ctx := context.Background()

	cases := map[string]func(taskapp.TaskStore) error{
		"done":     func(s taskapp.TaskStore) error { return s.SetStatus(ctx, "t1", taskdomain.StatusDone, time.Now()) },
		"failed":   func(s taskapp.TaskStore) error { return s.SetStatus(ctx, "t1", taskdomain.StatusFailed, time.Now()) },
		"archived": func(s taskapp.TaskStore) error { return s.Archive(ctx, "u1", "t1", time.Now()) },
		"deleted": func(s taskapp.TaskStore) error {
			_, err := s.DeleteOwned(ctx, "u1", []string{"t1"})

			return err //nolint:wrapcheck // a test
		},
		"project deleted": func(s taskapp.TaskStore) error { return s.DeleteProject(ctx, "u1", "p1") },
	}

	for name, end := range cases {
		db, dir := egressapptest.DB(t), t.TempDir()
		hub := egressapptest.Hub(t)
		rt := egressapptest.Start(t, db, hub, dir)
		store := egressruntime.Tasks(tasksqlite.NewStore(db, opaque.Generator{}), rt.TaskEnded)

		held, id := openTunnel(t, rt, "t1")
		other, otherID := openTunnel(t, rt, "t2")

		if _, err := hub.TaskDir(ctx, "ws", egressproxy.Scope{Project: "p1", Task: "t1"}); err != nil {
			t.Fatal(err)
		}

		if err := end(store); err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if !held.closed || rt.Registry().Has(id) || len(hub.TaskScopes()) != 0 {
			t.Errorf("%s: closed %v, held %v, sockets %v", name, held.closed, rt.Registry().Has(id), hub.TaskScopes())
		}

		if name != "project deleted" && (other.closed || !rt.Registry().Has(otherID)) {
			t.Errorf("%s: another task's grant was disturbed", name)
		}
	}
}

func TestAnOpenTasksStatusChangesLeaveItsGrantsAlone(t *testing.T) {
	ctx := context.Background()
	db, dir := egressapptest.DB(t), t.TempDir()
	rt := egressapptest.Start(t, db, egressapptest.Hub(t), dir)
	store := egressruntime.Tasks(tasksqlite.NewStore(db, opaque.Generator{}), rt.TaskEnded)
	held, id := openTunnel(t, rt, "t1")

	open := []taskdomain.Status{taskdomain.StatusReview, taskdomain.StatusNeedsInput, taskdomain.StatusWorking}
	for _, status := range open {
		if err := store.SetStatus(ctx, "t1", status, time.Now()); err != nil {
			t.Fatal(err)
		}
	}

	if held.closed || !rt.Registry().Has(id) {
		t.Fatal("a task that is still open keeps its grants")
	}
}
