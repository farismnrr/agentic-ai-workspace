package egressproxy

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

var errDown = errors.New("database is down")

type failing struct{}

func (failing) Verify(context.Context, string, Scope, string) error {
	return errDown
}

type spy struct {
	seen []string
	mu   sync.Mutex
}

func (s *spy) Verify(_ context.Context, id string, scope Scope, host string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seen = append(s.seen, id+"|"+scope.Project+"|"+scope.Task+"|"+host)

	return nil
}

func TestAFailingVerifierRefusesEverything(t *testing.T) {
	r := newRig(t, DefaultLimits(), failing{})
	r.allow("g1", scopeA, time.Minute)

	if c := r.open(scopeA, goodHost+":443"); c.status != http.StatusForbidden {
		t.Fatalf("%d", c.status)
	}

	if r.lookups.Load() != 0 {
		t.Fatal("nothing may be resolved when the source of truth cannot answer")
	}
}

func TestTheVerifierIsToldTheServersScopeNotTheClients(t *testing.T) {
	s := &spy{}
	r := newRig(t, DefaultLimits(), s)
	r.allow("g1", scopeA, time.Minute)

	established(t, r, scopeA)

	if len(s.seen) != 1 || s.seen[0] != "g1|p1|t1|"+goodHost {
		t.Fatalf("verifier saw %v", s.seen)
	}
}
