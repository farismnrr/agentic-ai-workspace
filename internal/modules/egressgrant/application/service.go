// Package application runs the egress grant use cases. It decides nothing about the network: it keeps requests and
// decisions, and says which hosts a task's sandbox is currently approved for.
package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

// Errors of the use cases. The store reports them, the service passes them on.
var (
	ErrNotFound  = errors.New("grant not found")
	ErrScope     = errors.New("that task does not belong to that project")
	ErrDuplicate = errors.New("that task already has a live request for that host")
	ErrConflict  = errors.New("the grant changed while it was being decided; try again")
)

// Store keeps grants. Every method is atomic: a decision either replaces exactly the state it was made from or
// changes nothing.
type Store interface {
	// Create saves a pending grant. It returns ErrScope unless the task belongs to the project, and ErrDuplicate when
	// the task already has a pending or active grant for the host.
	Create(ctx context.Context, grant domain.Grant) error
	Get(ctx context.Context, id string) (domain.Grant, error)
	// Decide loads the grant, lets change work out the next state and saves it only if nothing else changed it in
	// between, retrying a few times before it gives up with ErrConflict. An error from change is returned as it is.
	Decide(ctx context.Context, id string, change func(domain.Grant) (domain.Grant, error)) (domain.Grant, error)
	// Active lists the grants of one task that are approved and not past their end at now.
	Active(ctx context.Context, projectID, taskID string, now time.Time) ([]domain.Grant, error)
	List(ctx context.Context, projectID, taskID string) ([]domain.Grant, error)
	// TaskLive reports whether the task exists in that project and is still open: not archived, not done or failed.
	TaskLive(ctx context.Context, projectID, taskID string) (bool, error)
	// EndInactive ends every pending or approved grant whose task is no longer live, for good, and returns how many.
	EndInactive(ctx context.Context, now time.Time) (int, error)
	// LiveApproved lists every approved grant that has not ended and whose task is live: what the proxy should hold.
	LiveApproved(ctx context.Context, now time.Time) ([]domain.Grant, error)
}

// IDs makes opaque identifiers.
type IDs interface{ UUID() (string, error) }

// Service is the egress grant use cases.
type Service struct {
	store    Store
	ids      IDs
	registry Registry
	now      func() time.Time
	syncMu   sync.Mutex
}

// NewService builds the use cases. now is injectable for tests.
func NewService(store Store, ids IDs, now func() time.Time) *Service {
	return &Service{store: store, ids: ids, now: now}
}

// Request records that a task wants to reach host. An agent or a person can ask; nothing is allowed until a
// person decides.
func (s *Service) Request(
	ctx context.Context, projectID, taskID, host, reason string, by domain.Actor,
) (domain.Grant, error) {
	id, err := s.ids.UUID()
	if err != nil {
		return domain.Grant{}, fmt.Errorf("make grant id: %w", err)
	}

	grant, err := domain.Request(id, projectID, taskID, host, reason, by, s.now())
	if err != nil {
		return domain.Grant{}, fmt.Errorf("request grant: %w", err)
	}

	if err = s.store.Create(ctx, grant); err != nil {
		return domain.Grant{}, fmt.Errorf("save grant: %w", err)
	}

	return grant, nil
}

// Approve lets the task reach the host for ttl (a day when zero).
func (s *Service) Approve(ctx context.Context, id string, by domain.Decider, ttl time.Duration) (domain.Grant, error) {
	return s.decide(ctx, id, func(g domain.Grant) (domain.Grant, error) { return g.Approve(by, ttl, s.now()) })
}

// Deny turns a pending request down.
func (s *Service) Deny(ctx context.Context, id string, by domain.Decider) (domain.Grant, error) {
	return s.decide(ctx, id, func(g domain.Grant) (domain.Grant, error) { return g.Deny(by, s.now()) })
}

// Revoke ends an active grant now.
func (s *Service) Revoke(ctx context.Context, id string, by domain.Decider) (domain.Grant, error) {
	return s.decide(ctx, id, func(g domain.Grant) (domain.Grant, error) { return g.Revoke(by, s.now()) })
}

func (s *Service) decide(
	ctx context.Context, id string, change func(domain.Grant) (domain.Grant, error),
) (domain.Grant, error) {
	grant, err := s.store.Decide(ctx, id, change)
	if err != nil {
		return domain.Grant{}, fmt.Errorf("decide on grant: %w", err)
	}

	// The decision is committed. The enforcement point is brought in line with the database before the caller is
	// told; if that fails the caller hears about it, and enforcement stays closed (see syncGrant).
	if err = s.syncGrant(ctx, id); err != nil {
		return domain.Grant{}, err
	}

	return grant, nil
}

// Active lists the grants the task may use now. The domain rule is applied again after the store answers, so a
// store that returned too much would still not widen access.
func (s *Service) Active(ctx context.Context, projectID, taskID string) ([]domain.Grant, error) {
	now := s.now()

	found, err := s.store.Active(ctx, projectID, taskID, now)
	if err != nil {
		return nil, fmt.Errorf("list active grants: %w", err)
	}

	active := make([]domain.Grant, 0, len(found))

	for _, g := range found {
		if g.Covers(projectID, taskID, g.Host, now) {
			active = append(active, g)
		}
	}

	return active, nil
}

// List returns every grant of a task, newest first, for the person who decides on them.
func (s *Service) List(ctx context.Context, projectID, taskID string) ([]domain.Grant, error) {
	grants, err := s.store.List(ctx, projectID, taskID)
	if err != nil {
		return nil, fmt.Errorf("list grants: %w", err)
	}

	return grants, nil
}
