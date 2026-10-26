package sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

func TestConcurrentRequestsForOneHostCreateOne(t *testing.T) {
	w := newWorld(t)
	got := race(20, func(int) error {
		_, err := w.Request(context.Background(), "p1", "t1", "same.example", "why", agent)

		return only(err)
	})

	if got.ok != 1 || got.other != 0 || got.duplicate != 19 || w.count(t, "") != 1 {
		t.Fatalf("results %+v (first other: %v)", got, got.firstOther)
	}
}

func TestConcurrentApprovalsOfOneGrantSucceedOnce(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p1", "t1", "a.example")
	got := race(30, func(int) error {
		_, err := w.Approve(context.Background(), g.ID, p1own, time.Hour)

		return only(err)
	})

	if got.ok != 1 || got.other != 0 || got.notPending != 29 {
		t.Fatalf("results %+v (first other: %v)", got, got.firstOther)
	}
}

func TestApproveAndDenyRaceHasOneWinner(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p1", "t1", "a.example")
	got := race(20, func(i int) error {
		if i%2 == 0 {
			_, err := w.Approve(context.Background(), g.ID, p1own, time.Hour)

			return only(err)
		}

		_, err := w.Deny(context.Background(), g.ID, p1own)

		return only(err)
	})

	if got.ok != 1 || got.other != 0 || w.count(t, "WHERE status IN ('approved', 'denied')") != 1 {
		t.Fatalf("results %+v (first other: %v)", got, got.firstOther)
	}
}

func TestRevokeRacingApprovalsNeverLeavesAnActiveGrantAfterRevoke(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p1", "t1", "a.example")
	w.approve(t, g, p1own, time.Hour)

	got := race(20, func(i int) error {
		if i%2 == 0 {
			_, err := w.Revoke(context.Background(), g.ID, p1own)

			return only(err)
		}

		_, err := w.Approve(context.Background(), g.ID, p1own, domain.MaxTTL)

		return only(err)
	})

	if got.ok != 1 || got.other != 0 || got.notPending != 10 || got.notApproved != 9 ||
		w.count(t, "WHERE status = 'revoked'") != 1 {
		t.Fatalf("results %+v (first other: %v)", got, got.firstOther)
	}
}

func TestADecidedGrantCannotBeReplayed(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	g := w.ask(t, "p1", "t1", "a.example")
	w.approve(t, g, p1own, time.Hour)

	if _, err := w.Approve(ctx, g.ID, p1own, domain.MaxTTL); !errors.Is(err, domain.ErrNotPending) {
		t.Fatalf("a second approval must not stretch the life: %v", err)
	}

	if _, err := w.Revoke(ctx, g.ID, p1own); err != nil {
		t.Fatal(err)
	}

	*w.clock = t0.Add(time.Minute)

	if _, err := w.Approve(ctx, g.ID, p1own, time.Hour); !errors.Is(err, domain.ErrNotPending) {
		t.Errorf("approve after revoke: %v", err)
	}

	if _, err := w.Revoke(ctx, g.ID, p1own); !errors.Is(err, domain.ErrNotApproved) {
		t.Errorf("revoke twice: %v", err)
	}

	if got, _ := w.Active(ctx, "p1", "t1"); len(got) != 0 {
		t.Fatalf("still active: %+v", got)
	}
}
