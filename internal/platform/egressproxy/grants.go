package egressproxy

import (
	"errors"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

// Dynamic external grants. Everything in this file and grantpath.go is separate from the three older ways a
// sandbox gets out: the static registry list (Proxy.hosts), the exact internal targets (Proxy.targets) and the
// workspace's extra hosts. A grant can only be added through Grants.Allow, which the server calls after a person
// approved; the agent, the sandbox and the request never name the scope or the host to allow.

// Errors of the grant registry.
var (
	ErrBadGrant   = errors.New("that grant cannot be used")
	ErrNotGranted = errors.New("no live grant for that scope and host")
	ErrTooMany    = errors.New("too many open tunnels")
)

// Scope is whom a socket serves: one task of one project. The server builds it when it creates the socket and the
// listener keeps it; nothing the sandbox sends can change it.
type Scope struct{ Project, Task string }

func (s Scope) valid() bool { return s.Project != "" && s.Task != "" }

// Grants is the in-memory table of approved hosts per task, with the tunnels each one has open. It is the
// enforcement point: when a grant is revoked or ends, every tunnel bound to it is closed here.
type Grants struct {
	now     func() time.Time
	verify  Verifier
	byScope map[Scope]map[string]*grant
	byID    map[string]*grant
	open    map[Scope]int
	limits  Limits
	total   int
	mu      sync.Mutex
	closed  bool
}

type grant struct {
	ends  time.Time
	timer *time.Timer
	live  map[*Lease]struct{}
	id    string
	host  string
	scope Scope
}

// NewGrants builds an empty registry. verify may be nil.
func NewGrants(limits Limits, verify Verifier) *Grants {
	return &Grants{
		now: time.Now, limits: limits, verify: verify, open: map[Scope]int{},
		byScope: map[Scope]map[string]*grant{}, byID: map[string]*grant{},
	}
}

// Allow lets the scope reach host until ends. The host must already be a bare public name; it is stored in
// canonical form. A second grant for the same scope and host replaces the first and closes its tunnels.
func (g *Grants) Allow(id string, scope Scope, host string, ends time.Time) error {
	canonical, ok := egress.CanonicalHost(host)
	if id == "" || !scope.valid() || !ok {
		return ErrBadGrant
	}

	replaced, err := g.insert(id, scope, canonical, ends)

	// The replaced grant's tunnels are closed after the lock is released: closing a lease takes it again.
	closeAll(replaced)

	return err
}

// insert adds the entry under the lock and returns the tunnels of the grant it replaced.
func (g *Grants) insert(id string, scope Scope, host string, ends time.Time) ([]*Lease, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	wait := ends.Sub(g.now())
	if g.closed || wait <= 0 || g.byID[id] != nil {
		return nil, ErrBadGrant
	}

	var replaced []*Lease
	if old := g.byScope[scope][host]; old != nil {
		replaced = g.dropLocked(old)
	}

	entry := &grant{id: id, scope: scope, host: host, ends: ends, live: map[*Lease]struct{}{}}
	entry.timer = time.AfterFunc(wait, func() { g.Revoke(id) })

	if g.byScope[scope] == nil {
		g.byScope[scope] = map[string]*grant{}
	}

	g.byScope[scope][host], g.byID[id] = entry, entry

	return replaced, nil
}

// Revoke ends a grant now and closes every tunnel bound to it. When it returns, no tunnel of that grant is open
// and none can be opened. Unknown ids are ignored.
func (g *Grants) Revoke(id string) {
	g.mu.Lock()
	entry := g.byID[id]

	var leases []*Lease
	if entry != nil {
		leases = g.dropLocked(entry)
	}

	g.mu.Unlock()

	closeAll(leases)
}

// Clear revokes every grant of one task, for when the task is over.
func (g *Grants) Clear(scope Scope) {
	g.mu.Lock()

	var leases []*Lease
	for _, entry := range g.byScope[scope] {
		leases = append(leases, g.dropLocked(entry)...)
	}

	g.mu.Unlock()

	closeAll(leases)
}

// Close revokes everything and refuses new grants.
func (g *Grants) Close() {
	g.mu.Lock()
	g.closed = true

	var leases []*Lease
	for _, entry := range g.byID {
		leases = append(leases, g.dropLocked(entry)...)
	}

	g.mu.Unlock()

	closeAll(leases)
}

// OpenTunnels reports the number of currently leased tunnel slots.
func (g *Grants) OpenTunnels() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.total
}

// dropLocked removes an entry and returns its open tunnels, which the caller closes after unlocking.
func (g *Grants) dropLocked(entry *grant) []*Lease {
	entry.timer.Stop()
	delete(g.byID, entry.id)
	delete(g.byScope[entry.scope], entry.host)

	if len(g.byScope[entry.scope]) == 0 {
		delete(g.byScope, entry.scope)
	}

	leases := make([]*Lease, 0, len(entry.live))
	for lease := range entry.live {
		leases = append(leases, lease)
	}

	entry.live = nil

	return leases
}

func closeAll(leases []*Lease) {
	for _, lease := range leases {
		lease.Close()
	}
}
