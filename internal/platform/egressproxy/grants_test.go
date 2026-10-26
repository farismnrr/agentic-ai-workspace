package egressproxy

import (
	"errors"
	"testing"
	"time"
)

func (g *Grants) openCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.total
}

func newGrants(t *testing.T) *Grants {
	t.Helper()

	g := NewGrants(DefaultLimits(), nil)
	t.Cleanup(g.Close)

	return g
}

func TestAllowRefusesAnythingThatIsNotAWellFormedGrant(t *testing.T) {
	g := newGrants(t)
	soon := time.Now().Add(time.Minute)

	cases := map[string]error{
		"no id":        g.Allow("", scopeA, goodHost, soon),
		"no project":   g.Allow("a", Scope{Task: "t"}, goodHost, soon),
		"no task":      g.Allow("b", Scope{Project: "p"}, goodHost, soon),
		"ip":           g.Allow("c", scopeA, "10.0.0.1", soon),
		"wildcard":     g.Allow("d", scopeA, "*.example.com", soon),
		"with a port":  g.Allow("e", scopeA, goodHost+":443", soon),
		"already over": g.Allow("f", scopeA, goodHost, time.Now().Add(-time.Second)),
		"now":          g.Allow("h", scopeA, goodHost, time.Now()),
	}

	for name, err := range cases {
		if !errors.Is(err, ErrBadGrant) {
			t.Errorf("%s: %v", name, err)
		}
	}

	if g.Allow("ok", scopeA, goodHost, soon) != nil || !errors.Is(g.Allow("ok", scopeB, goodHost, soon), ErrBadGrant) {
		t.Error("a grant id can be used once")
	}
}

func TestLeaseNeedsAnExactLiveGrant(t *testing.T) {
	g := newGrants(t)
	_ = g.Allow("g1", scopeA, goodHost, time.Now().Add(time.Minute))

	for name, c := range map[string]struct {
		scope Scope
		host  string
	}{
		"other task": {scopeB, goodHost}, "other project": {scopeC, goodHost}, "other host": {scopeA, "x.example.com"},
		"subdomain": {scopeA, "a." + goodHost}, "zero scope": {Scope{}, goodHost},
		"not canonical": {scopeA, "SB.example.com"},
	} {
		if _, err := g.Lease(c.scope, c.host); !errors.Is(err, ErrNotGranted) {
			t.Errorf("%s: %v", name, err)
		}
	}

	lease, err := g.Lease(scopeA, goodHost)
	if err != nil || lease.ID() != "g1" {
		t.Fatal(err)
	}

	lease.Close()
}

func TestLeaseStopsAtTheEndEvenBeforeTheTimerFires(t *testing.T) {
	g := newGrants(t)
	_ = g.Allow("g1", scopeA, goodHost, time.Now().Add(time.Hour))

	g.now = func() time.Time { return time.Now().Add(2 * time.Hour) }

	if _, err := g.Lease(scopeA, goodHost); !errors.Is(err, ErrNotGranted) {
		t.Fatalf("a grant past its end must not open anything: %v", err)
	}
}

func TestRevokeClosesWhatTheLeaseHoldsAndFreesTheSlot(t *testing.T) {
	g := newGrants(t)
	_ = g.Allow("g1", scopeA, goodHost, time.Now().Add(time.Minute))

	lease, _ := g.Lease(scopeA, goodHost)
	a, b := &closer{}, &closer{}
	lease.Add(a)
	lease.Add(b)

	g.Revoke("g1")

	late := &closer{}
	lease.Add(late)

	if !a.closed.Load() || !b.closed.Load() || !late.closed.Load() || g.openCount() != 0 {
		t.Fatal("revoke must close every connection, including ones added afterwards, and free the slot")
	}

	if _, err := g.Lease(scopeA, goodHost); !errors.Is(err, ErrNotGranted) {
		t.Fatal(err)
	}

	g.Revoke("g1")
	g.Revoke("unknown")
}

func TestTheTimerRevokesAtTheEnd(t *testing.T) {
	g := newGrants(t)
	_ = g.Allow("g1", scopeA, goodHost, time.Now().Add(100*time.Millisecond))

	lease, _ := g.Lease(scopeA, goodHost)
	held := &closer{}
	lease.Add(held)

	eventually(t, "the grant to end", func() bool { return held.closed.Load() })

	if g.openCount() != 0 {
		t.Fatal("the slot must be free")
	}
}

func TestANewGrantForTheSameHostReplacesTheOldOne(t *testing.T) {
	g := newGrants(t)
	_ = g.Allow("old", scopeA, goodHost, time.Now().Add(time.Minute))

	lease, _ := g.Lease(scopeA, goodHost)
	held := &closer{}
	lease.Add(held)

	err := g.Allow("new", scopeA, goodHost, time.Now().Add(time.Minute))
	if err != nil || !held.closed.Load() {
		t.Fatalf("%v closed=%v", err, held.closed.Load())
	}

	if again, err := g.Lease(scopeA, goodHost); err != nil || again.ID() != "new" {
		t.Fatal(err)
	}
}

func TestClearAndCloseEndEverythingTheyCover(t *testing.T) {
	g := NewGrants(DefaultLimits(), nil)
	_ = g.Allow("a", scopeA, goodHost, time.Now().Add(time.Minute))
	_ = g.Allow("b", scopeB, goodHost, time.Now().Add(time.Minute))

	la, _ := g.Lease(scopeA, goodHost)
	lb, _ := g.Lease(scopeB, goodHost)
	ha, hb := &closer{}, &closer{}
	la.Add(ha)
	lb.Add(hb)

	g.Clear(scopeA)

	if !ha.closed.Load() || hb.closed.Load() {
		t.Fatal("clear covers one task only")
	}

	g.Close()

	if !hb.closed.Load() || !errors.Is(g.Allow("c", scopeA, goodHost, time.Now().Add(time.Minute)), ErrBadGrant) {
		t.Fatal("close ends everything and refuses new grants")
	}
}

func TestLimitsHoldAndSlotsAreReused(t *testing.T) {
	g := NewGrants(Limits{PerScope: 2, Total: 3, Handshake: time.Second, Idle: time.Second, Dial: time.Second}, nil)
	t.Cleanup(g.Close)

	for _, scope := range []Scope{scopeA, scopeB} {
		_ = g.Allow(scope.Task, scope, goodHost, time.Now().Add(time.Minute))
	}

	a1, _ := g.Lease(scopeA, goodHost)
	_, _ = g.Lease(scopeA, goodHost)

	if _, err := g.Lease(scopeA, goodHost); !errors.Is(err, ErrTooMany) {
		t.Fatalf("per task: %v", err)
	}

	_, _ = g.Lease(scopeB, goodHost)

	if _, err := g.Lease(scopeB, goodHost); !errors.Is(err, ErrTooMany) {
		t.Fatalf("in total: %v", err)
	}

	a1.Close()
	a1.Close()

	if _, err := g.Lease(scopeB, goodHost); err != nil {
		t.Fatalf("a freed slot is reusable: %v", err)
	}
}
