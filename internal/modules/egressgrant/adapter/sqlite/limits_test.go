package sqlite_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

func (w world) limited(max application.Limits) world {
	w.WithLimits(max)

	return w
}

func host(i int) string { return "h" + strconv.Itoa(i) + ".example" }

func TestOnlyAFewRequestsMayWaitPerTask(t *testing.T) {
	w := newWorld(t).limited(application.Limits{MaxPending: 3, MaxByAgent: 100, Window: time.Hour})
	ctx := context.Background()

	var asked []domain.Grant
	for i := range 3 {
		asked = append(asked, w.ask(t, "p1", "t1", host(i)))
	}

	if _, err := w.Request(ctx, "p1", "t1", host(9), "why", agent); !errors.Is(err, application.ErrTooManyPending) {
		t.Fatalf("fourth: %v", err)
	}

	w.ask(t, "p1", "t2", host(9))

	if _, err := w.Deny(ctx, asked[0].ID, p1own); err != nil {
		t.Fatal(err)
	}

	w.ask(t, "p1", "t1", host(9))
}

func TestAnAgentMayAskOnlySoOftenButAPersonIsNotCounted(t *testing.T) {
	w := newWorld(t).limited(application.Limits{MaxPending: 100, MaxByAgent: 4, Window: time.Hour})
	ctx := context.Background()

	for i := range 4 {
		g := w.ask(t, "p1", "t1", host(i))
		if _, err := w.Deny(ctx, g.ID, p1own); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := w.Request(ctx, "p1", "t1", host(9), "why", agent); !errors.Is(err, application.ErrRateLimited) {
		t.Fatalf("fifth in the hour, though none is waiting: %v", err)
	}

	person := domain.Actor{Kind: domain.ActorPerson, ID: "u1"}
	if _, err := w.Request(ctx, "p1", "t1", host(9), "why", person); err != nil {
		t.Fatalf("a person is not rate limited: %v", err)
	}

	*w.clock = t0.Add(time.Hour + time.Second)

	if _, err := w.Request(ctx, "p1", "t1", host(10), "why", agent); err != nil {
		t.Fatalf("after the window: %v", err)
	}
}

func TestConcurrentRequestsNeverExceedTheLimit(t *testing.T) {
	w := newWorld(t).limited(application.Limits{MaxPending: 5, MaxByAgent: 100, Window: time.Hour})
	got := race(30, func(i int) error {
		_, err := w.Request(context.Background(), "p1", "t1", host(i), "why", agent)

		return only(err)
	})

	if got.ok != 5 || got.limited != 25 || got.other != 0 || w.count(t, "") != 5 {
		t.Fatalf("results %+v", got)
	}
}

func TestAskingAgainReturnsTheSameRequest(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()

	first, existing, err := w.Ask(ctx, "p1", "t1", "Marimo.Example.", "need it", agent)
	if err != nil || existing {
		t.Fatalf("%v %v", err, existing)
	}

	again, existing, err := w.Ask(ctx, "p1", "t1", "marimo.example", "still", agent)
	if err != nil || !existing || again.ID != first.ID {
		t.Fatalf("%+v %v %v", again, existing, err)
	}

	w.approve(t, first, p1own, time.Hour)
	assertSame(t, w, first.ID)

	if w.count(t, "") != 1 {
		t.Fatal("one row only")
	}
}

func TestConcurrentAsksForOneHostAreOneRequest(t *testing.T) {
	w := newWorld(t)
	ids := make(chan string, 20)

	got := race(20, func(int) error {
		g, _, err := w.Ask(context.Background(), "p1", "t1", "same.example", "why", agent)
		ids <- g.ID

		return only(err)
	})

	close(ids)

	seen := map[string]bool{}
	for id := range ids {
		seen[id] = true
	}

	if got.ok != 20 || len(seen) != 1 || w.count(t, "") != 1 {
		t.Fatalf("results %+v, ids %v", got, seen)
	}
}

func TestAskRefusesWhatIsNotAnOpenTaskOrAHost(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	w.setStatus(t, "t2", "done")

	if _, _, err := w.Ask(ctx, "p1", "t2", "a.example", "why", agent); !errors.Is(err, application.ErrScope) {
		t.Errorf("a finished task: %v", err)
	}

	if _, _, err := w.Ask(ctx, "p1", "t3", "a.example", "why", agent); !errors.Is(err, application.ErrScope) {
		t.Errorf("another project's task: %v", err)
	}

	for _, bad := range []string{"127.0.0.1", "*.example.com", "a.example:443", "localhost", ""} {
		if _, _, err := w.Ask(ctx, "p1", "t1", bad, "why", agent); !errors.Is(err, domain.ErrInvalidHost) {
			t.Errorf("%q: %v", bad, err)
		}
	}

	if w.count(t, "") != 0 {
		t.Fatal("nothing may be saved")
	}
}

// assertSame asks again for the approved host and expects the very same, active grant back.
func assertSame(t *testing.T, w world, id string) {
	t.Helper()

	got, existing, err := w.Ask(context.Background(), "p1", "t1", "marimo.example", "again", agent)
	if err != nil || !existing || got.ID != id || !got.Active(*w.clock) {
		t.Fatalf("%+v %v %v", got, existing, err)
	}
}
