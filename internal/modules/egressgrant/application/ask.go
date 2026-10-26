package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

// Ask is Request for callers that may repeat themselves, such as an agent: asking again for a host that already has
// a live request returns that request instead of an error, so the answer is the same however often it is asked.
// existing says it was already there. The limits still apply to new requests.
func (s *Service) Ask(
	ctx context.Context, projectID, taskID, host, reason string, by domain.Actor,
) (grant domain.Grant, existing bool, err error) {
	const tries = 2

	for range tries {
		grant, err = s.Request(ctx, projectID, taskID, host, reason, by)
		if !errors.Is(err, ErrDuplicate) {
			return grant, false, err
		}

		normal, nerr := domain.NormalizeHost(host)
		if nerr != nil {
			return domain.Grant{}, false, fmt.Errorf("ask: %w", nerr)
		}

		found, lerr := s.store.Live(ctx, taskID, normal, s.now())
		if lerr == nil {
			return found, true, nil
		}

		if !errors.Is(lerr, ErrNotFound) {
			return domain.Grant{}, false, fmt.Errorf("find request: %w", lerr)
		}
	}

	return domain.Grant{}, false, err
}

// TaskEnded ends every grant of a task for good and closes its tunnels. The task loop calls it the moment a task is
// done, failed, archived or deleted; the sweep only repeats it as a safety net.
func (s *Service) TaskEnded(ctx context.Context, taskID string) error {
	if taskID == "" {
		return nil
	}

	var dbErr error

	if _, err := s.store.EndTask(ctx, taskID, s.now()); err != nil {
		dbErr = fmt.Errorf("end grants of task: %w", err)
	}

	// The registry is cleared whatever the database said: a task that is over keeps no access.
	if s.registry != nil {
		s.syncMu.Lock()
		s.registry.EndTask(taskID)
		s.syncMu.Unlock()
	}

	return dbErr
}
