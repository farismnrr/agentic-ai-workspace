package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

// ErrNotActive means a connection is not covered by a live grant of a live task.
var ErrNotActive = errors.New("no live grant covers that connection")

// Registry is the table of approved hosts the proxy enforces, held in memory by the process. The database is the
// source of truth; this port only mirrors it, and every method must be safe to call at any time from any goroutine.
type Registry interface {
	// Allow lets the task reach host until ends. It does nothing if the grant id is already held.
	Allow(id, projectID, taskID, host string, ends time.Time) error
	// Revoke ends a grant and closes every tunnel bound to it before it returns. Unknown ids are ignored.
	Revoke(id string)
	// IDs lists the grants currently held.
	IDs() []string
}

// WithRegistry makes decisions and sweeps keep reg in line with the database. Without it the service only keeps
// records, which is how the feature stays inert.
func (s *Service) WithRegistry(reg Registry) *Service {
	s.registry = reg

	return s
}

// syncGrant brings the registry in line with the database for one grant. It reads the state under a lock that
// every registry change takes, so two commits racing for the registry cannot leave an older state behind: the
// last one to run applies what the database says at that moment. Anything that goes wrong ends the grant in the
// registry.
func (s *Service) syncGrant(ctx context.Context, id string) error {
	if s.registry == nil {
		return nil
	}

	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	g, err := s.store.Get(ctx, id)
	if err != nil {
		s.registry.Revoke(id)

		return fmt.Errorf("read grant: %w", err)
	}

	live := g.Active(s.now())

	if live {
		if live, err = s.store.TaskLive(ctx, g.ProjectID, g.TaskID); err != nil {
			s.registry.Revoke(id)

			return fmt.Errorf("check task: %w", err)
		}
	}

	if !live {
		s.registry.Revoke(id)

		return nil
	}

	if err = s.registry.Allow(g.ID, g.ProjectID, g.TaskID, g.Host, g.ExpiresAt); err != nil {
		s.registry.Revoke(id)

		return fmt.Errorf("enforce grant: %w", err)
	}

	return nil
}

// Verify is the proxy's check on every connection that matched a grant. It asks the database, so a grant that was
// revoked, ended, or whose task is over or gone is refused at once, and so is any connection when the database
// cannot answer: the result is an error unless everything is confirmed.
func (s *Service) Verify(ctx context.Context, grantID, projectID, taskID, host string) error {
	g, err := s.store.Get(ctx, grantID)
	if err != nil {
		return fmt.Errorf("verify grant: %w", err)
	}

	if !g.Covers(projectID, taskID, host, s.now()) {
		return ErrNotActive
	}

	live, err := s.store.TaskLive(ctx, projectID, taskID)
	if err != nil {
		return fmt.Errorf("verify task: %w", err)
	}

	if !live {
		return ErrNotActive
	}

	return nil
}

// TaskLive tells whether the task exists in that project and is still open. The runner asks it before it gives a
// sandbox a task socket.
func (s *Service) TaskLive(ctx context.Context, projectID, taskID string) (bool, error) {
	live, err := s.store.TaskLive(ctx, projectID, taskID)
	if err != nil {
		return false, fmt.Errorf("check task: %w", err)
	}

	return live, nil
}

// Reconcile makes the registry equal to what the database says should be enforced: approved, not ended, task
// still open. It ends grants of finished tasks for good, closes tunnels of anything no longer allowed and (at
// start, or after a failure) restores the rest. It runs at start and then on a timer, which is also how a task
// that ended, was archived or was deleted loses its grants. If the database cannot be read, everything is revoked
// and rebuilt on the next successful run.
func (s *Service) Reconcile(ctx context.Context) error {
	if s.registry == nil {
		return nil
	}

	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	now := s.now()

	rows, err := s.wanted(ctx, now)
	if err != nil {
		for _, id := range s.registry.IDs() {
			s.registry.Revoke(id)
		}

		return err
	}

	want := make(map[string]domain.Grant, len(rows))
	for _, g := range rows {
		want[g.ID] = g
	}

	for _, id := range s.registry.IDs() {
		if _, ok := want[id]; !ok {
			s.registry.Revoke(id)
		}
	}

	var first error

	for _, g := range rows {
		if err = s.registry.Allow(g.ID, g.ProjectID, g.TaskID, g.Host, g.ExpiresAt); err != nil && first == nil {
			first = fmt.Errorf("enforce grant: %w", err)
		}
	}

	return first
}

func (s *Service) wanted(ctx context.Context, now time.Time) ([]domain.Grant, error) {
	if _, err := s.store.EndInactive(ctx, now); err != nil {
		return nil, fmt.Errorf("sweep: %w", err)
	}

	rows, err := s.store.LiveApproved(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("sweep: %w", err)
	}

	return rows, nil
}
