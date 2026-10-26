package domain

import (
	"errors"
	"testing"
	"time"
)

var (
	t0     = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	agent  = Actor{Kind: ActorAgent, ID: "agent:t1"}
	owner  = Decider{ID: "u1", ProjectID: "p1", Owner: true}
	admin  = Decider{ID: "u2", Admin: true}
	nobody = Decider{ID: "u3", ProjectID: "p1"}
)

func pending(t *testing.T) Grant {
	t.Helper()

	g, err := Request("g1", "p1", "t1", "Marimo.Example.", "open the notebook", agent, t0)
	if err != nil {
		t.Fatal(err)
	}

	return g
}

func TestRequestStartsPendingAndStoresTheNormalizedHost(t *testing.T) {
	g := pending(t)

	if g.Status != StatusPending || g.Host != "marimo.example" || g.Active(t0) {
		t.Fatalf("unexpected grant %+v", g)
	}

	if !g.ExpiresAt.Equal(t0.Add(PendingTTL)) {
		t.Fatalf("a pending request ends after %v, got %v", PendingTTL, g.ExpiresAt)
	}
}

func TestRequestRefusesBadInput(t *testing.T) {
	cases := map[string][5]string{
		"no id":      {"", "p", "t", "a.example", "why"},
		"no project": {"g", "", "t", "a.example", "why"},
		"no task":    {"g", "p", "", "a.example", "why"},
		"no reason":  {"g", "p", "t", "a.example", "  "},
		"long":       {"g", "p", "t", "a.example", string(make([]byte, maxReasonLength+1))},
	}

	for name, c := range cases {
		if _, err := Request(c[0], c[1], c[2], c[3], c[4], agent, t0); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: got %v", name, err)
		}
	}

	for _, who := range []Actor{{Kind: ActorAgent}, {ID: "x"}, {Kind: "robot", ID: "x"}} {
		if _, err := Request("g", "p", "t", "a.example", "why", who, t0); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("actor %+v: got %v", who, err)
		}
	}

	if _, err := Request("g", "p", "t", "10.0.0.1", "why", agent, t0); !errors.Is(err, ErrInvalidHost) {
		t.Errorf("an IP address must be refused, got %v", err)
	}
}

func TestApprovalLastsADayByDefaultAndStopsThen(t *testing.T) {
	g, err := pending(t).Approve(owner, 0, t0)
	if err != nil {
		t.Fatal(err)
	}

	if !g.ExpiresAt.Equal(t0.Add(DefaultTTL)) || g.DecidedBy != "u1" || !g.DecidedAt.Equal(t0) {
		t.Fatalf("unexpected grant %+v", g)
	}

	if !g.Active(t0.Add(DefaultTTL - time.Second)) {
		t.Error("must be active just before the end")
	}

	for _, at := range []time.Time{t0.Add(DefaultTTL), t0.Add(DefaultTTL + time.Hour)} {
		if g.Active(at) || g.Effective(at) != StatusExpired {
			t.Errorf("must be expired at %v", at)
		}
	}
}

func TestLifetimeIsCapped(t *testing.T) {
	g := pending(t)

	if _, err := g.Approve(owner, MaxTTL+time.Second, t0); !errors.Is(err, ErrTTLTooLong) {
		t.Errorf("over the cap: %v", err)
	}

	if _, err := g.Approve(owner, -time.Hour, t0); !errors.Is(err, ErrTTLTooLong) {
		t.Errorf("negative: %v", err)
	}

	if a, err := g.Approve(owner, MaxTTL, t0); err != nil || !a.ExpiresAt.Equal(t0.Add(MaxTTL)) {
		t.Errorf("the cap itself is allowed: %v", err)
	}
}

func TestCoversIsOneTaskOneHostAndNothingElse(t *testing.T) {
	g, _ := pending(t).Approve(owner, 0, t0)
	now := t0.Add(time.Minute)

	if !g.Covers("p1", "t1", "MARIMO.example.", now) {
		t.Error("the granted task and host, however spelled, must be covered")
	}

	no := map[string][3]string{
		"other task":      {"p1", "t2", "marimo.example"},
		"other project":   {"p2", "t1", "marimo.example"},
		"other host":      {"p1", "t1", "other.example"},
		"subdomain":       {"p1", "t1", "a.marimo.example"},
		"parent":          {"p1", "t1", "example"},
		"with a port":     {"p1", "t1", "marimo.example:443"},
		"an ip":           {"p1", "t1", "127.0.0.1"},
		"empty":           {"p1", "t1", ""},
		"suffix trickery": {"p1", "t1", "xmarimo.example"},
	}

	for name, c := range no {
		if g.Covers(c[0], c[1], c[2], now) {
			t.Errorf("%s must not be covered", name)
		}
	}

	if g.Covers("p1", "t1", "marimo.example", t0.Add(DefaultTTL)) {
		t.Error("an expired grant covers nothing")
	}

	if pending(t).Covers("p1", "t1", "marimo.example", t0) {
		t.Error("a pending request covers nothing")
	}
}
