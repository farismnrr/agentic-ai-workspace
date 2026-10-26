package egressproxy

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func established(t *testing.T, r *rig, scope Scope) *client {
	t.Helper()

	c := r.open(scope, goodHost+":443")
	if c.status != http.StatusOK || !c.echoes(helloFor(t, goodHost)) {
		t.Fatalf("tunnel not established: %d", c.status)
	}

	return c
}

func TestAHelloForAnotherNameIsClosedBeforeTheDestinationSeesIt(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)

	c := r.open(scopeA, goodHost+":443")
	_, _ = c.Write(helloFor(t, "other.example.com"))

	if c.status != http.StatusOK || !c.closedWithin(time.Second) || r.received.len() != 0 {
		t.Fatalf("status %d, destination saw %d bytes", c.status, r.received.len())
	}
}

func TestNonTLSFirstBytesAreClosedBeforeTheDestinationSeesThem(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)

	for name, first := range map[string][]byte{
		"http": []byte("GET / HTTP/1.1\r\nHost: " + goodHost + "\r\n\r\n"), "ssh": []byte("SSH-2.0-x\r\n"),
		"garbage": make([]byte, 64), "alert record": {21, 3, 3, 0, 2, 2, 40},
	} {
		c := r.open(scopeA, goodHost+":443")
		_, _ = c.Write(first)

		if !c.closedWithin(time.Second) {
			t.Errorf("%s: the tunnel stayed open", name)
		}
	}

	if r.received.len() != 0 {
		t.Fatalf("the destination saw %d bytes", r.received.len())
	}
}

func TestAClientThatSaysNothingIsClosedAtTheHandshakeLimit(t *testing.T) {
	limits := DefaultLimits()
	limits.Handshake = 150 * time.Millisecond
	r := newRig(t, limits, nil)
	r.allow("g1", scopeA, time.Minute)

	c := r.open(scopeA, goodHost+":443")
	if !c.closedWithin(2 * time.Second) {
		t.Fatal("a silent client must not hold a tunnel open")
	}

	eventually(t, "the slot to be freed", func() bool { return r.grants.openCount() == 0 })
}

func TestAnIdleTunnelIsClosed(t *testing.T) {
	limits := DefaultLimits()
	limits.Idle = 200 * time.Millisecond
	r := newRig(t, limits, nil)
	r.allow("g1", scopeA, time.Minute)

	c := established(t, r, scopeA)
	if !c.closedWithin(2 * time.Second) {
		t.Fatal("an idle tunnel must end")
	}
}

func TestRevokeClosesTheOpenTunnelAndBlocksNewOnes(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)
	c := established(t, r, scopeA)

	r.grants.Revoke("g1")

	if !c.closedWithin(time.Second) {
		t.Fatal("the tunnel must be closed by the revoke")
	}

	if again := r.open(scopeA, goodHost+":443"); again.status != http.StatusForbidden {
		t.Fatalf("after revoke: %d", again.status)
	}

	eventually(t, "the slot to be freed", func() bool { return r.grants.openCount() == 0 })
}

func TestExpiryClosesTheOpenTunnelAndBlocksNewOnes(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, 400*time.Millisecond)
	c := established(t, r, scopeA)

	if !c.closedWithin(3 * time.Second) {
		t.Fatal("the tunnel must end with its grant")
	}

	if again := r.open(scopeA, goodHost+":443"); again.status != http.StatusForbidden {
		t.Fatalf("after expiry: %d", again.status)
	}
}

func TestTheOpenTunnelLimitPerTaskHolds(t *testing.T) {
	limits := DefaultLimits()
	limits.PerScope = 2
	r := newRig(t, limits, nil)
	r.allow("g1", scopeA, time.Minute)
	r.allow("g2", scopeB, time.Minute)

	established(t, r, scopeA)
	established(t, r, scopeA)

	if c := r.open(scopeA, goodHost+":443"); c.status != http.StatusServiceUnavailable {
		t.Fatalf("third tunnel of one task: %d", c.status)
	}

	established(t, r, scopeB)
}

func TestRevokingOneTasksGrantLeavesOtherTunnelsAlone(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("ga", scopeA, time.Minute)
	r.allow("gb", scopeB, time.Minute)

	const each = 8

	var (
		wg sync.WaitGroup
		mu sync.Mutex
		as []*client
		bs []*client
	)

	for range each {
		wg.Add(2)

		go func() { defer wg.Done(); c := established(t, r, scopeA); mu.Lock(); as = append(as, c); mu.Unlock() }()
		go func() { defer wg.Done(); c := established(t, r, scopeB); mu.Lock(); bs = append(bs, c); mu.Unlock() }()
	}

	wg.Wait()
	r.grants.Revoke("ga")

	for i, c := range as {
		if !c.closedWithin(2 * time.Second) {
			t.Errorf("task A tunnel %d survived the revoke", i)
		}
	}

	for i, c := range bs {
		if !c.echoes([]byte("still here")) {
			t.Errorf("task B tunnel %d was disturbed", i)
		}
	}
}
