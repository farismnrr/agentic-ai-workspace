package application

import (
	"context"
	"errors"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// externalParallelCheck is called while externalMu is held. Worker tool
// boundaries remain enforced by the existing child runner; this protects the
// shared task/worktree from overlapping mutating workers.
func (s *Service) externalParallelCheck(ctx context.Context, r *Runner, task domain.Task, roster *Roster, requested string) error {
	desired, ok := roster.member(requested)
	if !ok {
		return domain.ErrBadRole
	}
	count := 0
	for key := range s.externalRunning {
		if !strings.HasPrefix(key, task.ID+"/") {
			continue
		}
		count++
		if count >= 4 {
			return errors.New("too many concurrent jobs on this task")
		}
		id := strings.TrimPrefix(key, task.ID+"/")
		existing, found, err := r.deps.Delegations.FindDelegation(ctx, task.ID, id)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("active worker state unavailable")
		}
		role, ok := roster.member(existing.RoleKey)
		if !ok {
			return errors.New("active worker role unavailable")
		}
		if roleCanWrite(role) && roleCanWrite(desired) {
			// Persisted delegation records do not store separate write scopes;
			// serialize write-capable jobs until a durable scope model exists.
			return errors.New("another file-changing worker is active; finish or cancel it first")
		}
		if !canParallel(role) || !canParallel(desired) {
			return errors.New("this worker must run on its own")
		}
	}
	return nil
}
