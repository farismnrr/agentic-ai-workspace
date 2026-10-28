package application

import (
	"context"
	"encoding/json"
)

// callDelegate runs one delegation from a CLI lead, allowing safe parallel delegations while serializing shared state.
func (l *leadSession) callDelegate(ctx context.Context, input json.RawMessage) (string, bool) {
	id, call, d, text, failed, ready := l.beginLeadDelegate(ctx, input)
	if !ready {
		return text, failed
	}

	out, done := l.inline.runDelegation(ctx, l.task, l.state, call, d)

	return l.finishLeadDelegate(id, out, done)
}

func (l *leadSession) beginLeadDelegate(
	ctx context.Context, input json.RawMessage,
) (string, Call, Delegation, string, bool, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.stopped != nil {
		return "", Call{}, Delegation{}, "Stop now: the run is over for this turn.", true, false
	}

	if why := l.inline.refusal(toolDelegate); why != "" {
		return "", Call{}, Delegation{}, why, true, false
	}

	id := l.callID(ctx, toolDelegate, input)
	call := Call{ID: id, Name: toolDelegate, Input: input}

	args, ok := parseDelegate(input)
	if !ok || l.state.roster == nil {
		return "", Call{}, Delegation{},
			"delegate needs a role and a brief, and works only on a task with a team.", true, false
	}

	d, found, err := l.inline.deps.Delegations.FindDelegation(ctx, l.task.ID, id)
	if err != nil {
		out := failed("I lost track of the team's work.", "Try Continue in a moment.")
		l.stopped = &out

		return "", Call{}, Delegation{}, stopText(out), true, false
	}

	if found && (d.Status == DelegationDone || d.Status == DelegationFailed) {
		return "", Call{}, Delegation{}, d.Report, d.Status == DelegationFailed, false
	}

	return l.activateLeadDelegate(ctx, id, call, args, d, found)
}

func (l *leadSession) activateLeadDelegate(
	ctx context.Context, id string, call Call, args delegateArgs, d Delegation, found bool,
) (string, Call, Delegation, string, bool, bool) {
	role, ok := l.state.roster.member(args.Role)
	if !ok {
		return "", Call{}, Delegation{},
			"There is no teammate called " + args.Role + ". " + teammateKeys(l.state.roster), true, false
	}

	if why := l.leadParallelConflict(role, args.Scope); why != "" {
		return "", Call{}, Delegation{}, why, true, false
	}

	if !found {
		if d, ok = l.inline.startDelegation(ctx, l.task, l.state, call, args); !ok {
			got := l.results.take(id)
			return "", Call{}, Delegation{}, got.text, got.failed, false
		}
	} else {
		d.Scope = args.Scope
	}

	l.active[id] = activeLeadJob{role: role, scope: d.Scope}

	return id, call, d, "", false, true
}

func (l *leadSession) leadParallelConflict(role AgentRole, scope []string) string {
	if len(l.active) == 0 {
		return ""
	}

	if !canParallel(role) {
		return parallelRoleRefused(role)
	}

	var (
		otherWrites  int
		activeScopes [][]string
	)

	for _, job := range l.active {
		if !canParallel(job.role) {
			return parallelRoleRefused(job.role)
		}

		if roleCanWrite(job.role) {
			otherWrites++
			if len(job.scope) == 0 && roleCanWrite(role) {
				return "Parallel write-capable teammates need a non-empty scope array of relative file or folder paths " +
					"so their edits do not collide."
			}
			activeScopes = append(activeScopes, job.scope)
		}
	}

	return parallelScopeConflict(role, scope, otherWrites, activeScopes)
}

func (l *leadSession) finishLeadDelegate(id string, out outcome, done bool) (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.active, id)

	if done {
		if l.stopped == nil || out.stop == stopFailed {
			l.stopped = &out
		}

		return stopText(out), out.stop == stopFailed
	}

	got := l.results.take(id)

	return got.text, got.failed
}
