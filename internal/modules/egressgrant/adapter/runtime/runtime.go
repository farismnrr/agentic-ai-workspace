package runtime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	egresssqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/sqlite"
	egressapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/instancelock"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

const (
	egressSweep   = 5 * time.Second
	egressTimeout = 5 * time.Second
	lockName      = ".grants.lock"
	egressDirMode = 0o750
)

// egressRuntime is dynamic egress grants wired into the running app. It exists only when EGRESS_GRANTS is on.
type Runtime struct {
	db      *sql.DB
	service *egressapp.Service
	grants  *egressproxy.Grants
	hub     *egressproxy.Hub
	audit   *egresssqlite.Audit
	stop    func()
}

// startEgressGrants wires the grant registry into the hub, restores what the database says is still valid, and
// starts the sweep that keeps the registry equal to the database. The registry lives in this process, so it takes
// a lock in the data folder first: a second instance cannot start it, and without it that instance has no grants
// at all (fail closed) instead of a registry that would miss revocations made elsewhere.
func Start(db *sql.DB, hub *egressproxy.Hub, dataDir string) (*Runtime, error) {
	dir := filepath.Join(dataDir, "egress")
	if err := os.MkdirAll(dir, egressDirMode); err != nil {
		return nil, fmt.Errorf("create egress folder: %w", err)
	}

	release, err := instancelock.Acquire(filepath.Join(dir, lockName))
	if err != nil {
		if errors.Is(err, instancelock.ErrHeld) {
			return nil, fmt.Errorf("egress grants need a single instance per data folder: %w", err)
		}

		return nil, fmt.Errorf("lock egress grants: %w", err)
	}

	verifier := &egressVerifier{}
	grants := egressproxy.NewGrants(egressproxy.DefaultLimits(), verifier)
	audit := egresssqlite.NewAudit(db)
	service := egressapp.NewService(egresssqlite.NewStore(db), opaque.Generator{}, time.Now).
		WithRegistry(egressRegistry{grants, audit})
	verifier.service = service

	hub.WithGrants(grants)

	rt := &Runtime{db: db, service: service, grants: grants, hub: hub, audit: audit}
	rt.stop = rt.run(release)

	return rt, nil
}

// run restores the registry, then sweeps until stopped.
func (rt *Runtime) run(release func()) func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	rt.sweep(ctx)

	go func() {
		defer close(done)

		tick := time.NewTicker(egressSweep)
		defer tick.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				rt.sweep(ctx)
			}
		}
	}()

	return func() {
		cancel()
		<-done
		rt.grants.Close()
		release()
	}
}

// sweep brings the registry in line with the database and stops the sockets of tasks that are over.
func (rt *Runtime) sweep(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, egressTimeout)
	defer cancel()

	if err := rt.service.Reconcile(ctx); err != nil {
		log.Printf("egress grants: %v", err)

		return
	}

	if _, err := rt.audit.Expired(ctx, time.Now()); err != nil {
		log.Printf("egress grants: expiration audit unavailable")
	}
	for _, scope := range rt.hub.TaskScopes() {
		if live, err := rt.service.TaskLive(ctx, scope.Project, scope.Task); err == nil && !live {
			rt.hub.RemoveTask(scope)
		}
	}
}

// Sweep synchronizes proxy grants and task sockets against persisted grant state.
func (rt *Runtime) Sweep(ctx context.Context) { rt.sweep(ctx) }

// egressRegistry lets the application keep the proxy's in-memory grants in line with the database.
type egressRegistry struct {
	grants *egressproxy.Grants
	audit  *egresssqlite.Audit
}

func (r egressRegistry) Allow(id, projectID, taskID, host string, ends time.Time) error {
	if r.grants.Has(id) {
		return nil
	}

	scope := egressproxy.Scope{Project: projectID, Task: taskID}

	err := r.grants.Allow(id, scope, host, ends)
	if err != nil && r.audit != nil {
		ctx, cancel := context.WithTimeout(context.Background(), egressTimeout)
		defer cancel()
		if auditErr := r.audit.EnforcementFailure(ctx, id); auditErr != nil {
			log.Printf("egress grants: enforcement audit unavailable")
		}
	}
	return err //nolint:wrapcheck // worded by the caller
}

func (r egressRegistry) Revoke(id string) { r.grants.Revoke(id) }

func (r egressRegistry) IDs() []string { return r.grants.IDs() }

func (r egressRegistry) EndTask(taskID string) { r.grants.ClearTask(taskID) }

// egressVerifier is the proxy's per-connection check, answered by the database.
type egressVerifier struct{ service *egressapp.Service }

func (v *egressVerifier) Verify(ctx context.Context, grantID string, scope egressproxy.Scope, host string) error {
	if v.service == nil {
		return egressapp.ErrNotActive
	}

	ctx, cancel := context.WithTimeout(ctx, egressTimeout)
	defer cancel()

	return v.service.Verify(ctx, grantID, scope.Project, scope.Task, host) //nolint:wrapcheck // worded by the service
}

// TaskEnded closes everything a task had the moment it ends: its grants are ended for good and their tunnels
// closed, and its sockets stop. The sweep repeats this as a safety net; this is the main path.
func (rt *Runtime) TaskEnded(ctx context.Context, taskID string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), egressTimeout)
	defer cancel()

	if err := rt.service.TaskEnded(ctx, taskID); err != nil {
		log.Printf("egress grants: %v", err)
	}

	for _, scope := range rt.hub.TaskScopes() {
		if scope.Task == taskID {
			rt.hub.RemoveTask(scope)
		}
	}
}

// Service is the application boundary used by task tools and HTTP handlers.
func (rt *Runtime) Service() *egressapp.Service { return rt.service }

// Registry exposes proxy grant enforcement to adapters that wire task-scoped sockets.
func (rt *Runtime) Registry() *egressproxy.Grants { return rt.grants }

// Close stops the registry sweep and releases the single-instance lock.
func (rt *Runtime) Close() { rt.stop() }
