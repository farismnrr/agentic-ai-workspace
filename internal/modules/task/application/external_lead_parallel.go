package application

import (
	"context"
	"errors"
	"fmt"
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
			return refusal("CONCURRENCY_LIMIT", "Too many concurrent jobs (max 4 concurrent workers allowed). Wait for active workers to complete or cancel one.")
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
			return refusal("MUTATING_WORKER_ACTIVE", fmt.Sprintf("Worker %q (%s) is actively modifying files. Mutating workers cannot run in parallel; finish or cancel job %s first.", role.Name, role.Key, id))
		}
		if !canParallel(role) || !canParallel(desired) {
			return refusal("SOLO_WORKER_ACTIVE", fmt.Sprintf("Worker %q (%s) requires exclusive execution. Wait for job %s to finish or cancel it first.", role.Name, role.Key, id))
		}
	}
	return nil
}
