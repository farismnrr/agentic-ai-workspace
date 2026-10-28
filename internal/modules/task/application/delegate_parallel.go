package application

import (
	"context"
	"sync"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type preparedDelegation struct {
	call       Call
	delegation Delegation
}

// delegateBatchLen returns how many consecutive delegate tool calls start at pending[0].
func delegateBatchLen(pending []Call) int {
	count := 0
	for count < len(pending) && pending[count].Name == toolDelegate {
		count++
	}

	return count
}

// processDelegations runs a batch of delegate calls concurrently after validating their scopes and preconditions.
func (r *Runner) processDelegations(
	ctx context.Context, task domain.Task, state *runState, calls []Call,
) (outcome, bool) {
	if len(calls) == 1 {
		return r.delegate(ctx, task, state, calls[0])
	}

	if why := r.refusal(toolDelegate); why != "" {
		for _, call := range calls {
			r.result(ctx, task, call.ID, why, true)
		}

		return outcome{}, false
	}

	ready, aborted, ok := r.prepareDelegationBatch(ctx, task, state, calls)
	if !ok {
		return aborted, true
	}

	if len(ready) == 0 {
		return outcome{}, false
	}

	if len(ready) == 1 {
		return r.runDelegation(ctx, task, state, ready[0].call, ready[0].delegation)
	}

	return r.runDelegationBatch(ctx, task, state, ready)
}

func (r *Runner) prepareDelegationBatch(
	ctx context.Context, task domain.Task, state *runState, calls []Call,
) ([]preparedDelegation, outcome, bool) {
	writeCount := r.countWriteCalls(state, calls)
	otherWrites := max(writeCount-1, 0)

	var (
		ready        []preparedDelegation
		activeScopes [][]string
	)

	for _, call := range calls {
		item, run, out, stopNow := r.prepareOneDelegation(ctx, task, state, call, otherWrites, &activeScopes)
		if stopNow {
			return nil, out, false
		}
		if run {
			ready = append(ready, item)
		}
	}

	return ready, outcome{}, true
}

func (r *Runner) prepareOneDelegation(
	ctx context.Context, task domain.Task, state *runState, call Call, otherWrites int, activeScopes *[][]string,
) (preparedDelegation, bool, outcome, bool) {
	args, ok := parseDelegate(call.Input)
	if !ok || state.roster == nil {
		r.result(ctx, task, call.ID, "delegate needs a role and a brief, and works only on a task with a team.", true)
		return preparedDelegation{}, false, outcome{}, false
	}

	role, ok := state.roster.member(args.Role)
	if !ok {
		r.result(ctx, task, call.ID, "There is no teammate called "+args.Role+". "+teammateKeys(state.roster), true)
		return preparedDelegation{}, false, outcome{}, false
	}

	if why := parallelScopeConflict(role, args.Scope, otherWrites, *activeScopes); why != "" {
		r.result(ctx, task, call.ID, why, true)
		return preparedDelegation{}, false, outcome{}, false
	}

	d, ok, out, stop := r.findOrStartDelegation(ctx, task, state, call, args)
	if stop || !ok {
		return preparedDelegation{}, false, out, stop
	}

	if roleCanWrite(role) {
		*activeScopes = append(*activeScopes, d.Scope)
	}

	return preparedDelegation{call: call, delegation: d}, true, outcome{}, false
}

func (r *Runner) findOrStartDelegation(
	ctx context.Context, task domain.Task, state *runState, call Call, args delegateArgs,
) (Delegation, bool, outcome, bool) {
	d, found, err := r.deps.Delegations.FindDelegation(ctx, task.ID, call.ID)
	if err != nil {
		return Delegation{}, false, failed("I lost track of the team's work.", "Try Continue in a moment."), true
	}

	if found && (d.Status == DelegationDone || d.Status == DelegationFailed) {
		r.result(ctx, task, call.ID, d.Report, d.Status == DelegationFailed)
		return Delegation{}, false, outcome{}, false
	}

	if !found {
		var ok bool
		d, ok = r.startDelegation(ctx, task, state, call, args)
		return d, ok, outcome{}, false
	}

	d.Scope = args.Scope

	return d, true, outcome{}, false
}

func (r *Runner) countWriteCalls(state *runState, calls []Call) int {
	if state == nil || state.roster == nil {
		return 0
	}

	count := 0
	for _, call := range calls {
		if args, ok := parseDelegate(call.Input); ok {
			if role, exists := state.roster.member(args.Role); exists && canParallel(role) && roleCanWrite(role) {
				count++
			}
		}
	}

	return count
}

func (r *Runner) runDelegationBatch(
	ctx context.Context, task domain.Task, state *runState, ready []preparedDelegation,
) (outcome, bool) {
	type itemOutcome struct {
		out  outcome
		done bool
	}

	outcomes := make([]itemOutcome, len(ready))
	var wg sync.WaitGroup

	for i, item := range ready {
		wg.Add(1)
		go func(idx int, prep preparedDelegation) {
			defer wg.Done()
			out, done := r.runDelegation(ctx, task, state, prep.call, prep.delegation)
			outcomes[idx] = itemOutcome{out: out, done: done}
		}(i, item)
	}

	wg.Wait()

	var waiting *outcome
	for _, res := range outcomes {
		if !res.done {
			continue
		}
		if res.out.stop == stopFailed {
			return res.out, true
		}
		if res.out.stop == stopWaiting && waiting == nil {
			out := res.out
			waiting = &out
		}
	}

	if waiting != nil {
		_ = r.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusNeedsInput, r.deps.Now())
		return *waiting, true
	}

	return outcome{}, false
}
