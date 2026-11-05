package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// externalParallelCheck is called while externalMu is held. Worker tool
// boundaries remain enforced by the existing child runner; this protects the
// shared task/worktree from overlapping mutating workers.
func (s *Service) externalParallelCheck(
	ctx context.Context, r *Runner, task domain.Task, roster *Roster, requested, excludeID string,
) error {
	desired, ok := roster.member(requested)
	if !ok {
		return domain.ErrBadRole
	}
	reader, ok := r.deps.Delegations.(interface {
		ListDelegations(context.Context, string) ([]Delegation, error)
	})
	if !ok {
		return errors.New("durable worker state unavailable")
	}
	jobs, err := reader.ListDelegations(ctx, task.ID)
	if err != nil {
		return err
	}
	count := 0
	for _, existing := range jobs {
		if existing.ID == excludeID || (existing.Status != DelegationWorking && existing.Status != DelegationWaiting) {
			continue
		}
		count++
		if count >= 4 {
			return refusal("CONCURRENCY_LIMIT", "Too many concurrent jobs (max 4 concurrent workers allowed). Wait for active workers to complete or cancel one.")
		}
		role, ok := roster.member(existing.RoleKey)
		if !ok {
			return refusal("ROLE_NOT_ALLOWED", "An active worker's role is no longer enabled. Cancel that job before assigning another.")
		}
		if roleCanWrite(role) && roleCanWrite(desired) {
			return refusal("MUTATING_WORKER_ACTIVE", fmt.Sprintf("Worker %q (%s) is actively modifying files. Mutating workers cannot run in parallel; finish or cancel job %s first.", role.Name, role.Key, existing.ID))
		}
		if !canParallel(role) || !canParallel(desired) {
			return refusal("SOLO_WORKER_ACTIVE", fmt.Sprintf("Worker %q (%s) requires exclusive execution. Wait for job %s to finish or cancel it first.", role.Name, role.Key, existing.ID))
		}
	}
	return nil
}
