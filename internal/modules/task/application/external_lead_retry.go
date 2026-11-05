package application

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func canonicalExternalInput(in ExternalAgentInput) (ExternalAgentInput, bool) {
	in.AgentKey = canonicalMemberKey(strings.TrimSpace(in.AgentKey))
	in.Brief = truncate(strings.TrimSpace(in.Brief), maxBrief)
	in.Context = strings.TrimSpace(in.Context)
	in.Effort = strings.TrimSpace(in.Effort)
	if !validScope(in.Scope) {
		return in, false
	}
	in.Scope = cleanScope(in.Scope)
	slices.Sort(in.Scope)
	return in, true
}

func sameExternalInput(d Delegation, in ExternalAgentInput) bool {
	return d.RoleKey == in.AgentKey && d.Brief == in.Brief && d.Context == in.Context &&
		d.RequestEffort == in.Effort && slices.Equal(d.Scope, in.Scope)
}

func (s *Service) retryExternal(
	ctx context.Context, userID string, task domain.Task, r *Runner, in ExternalAgentInput,
	jobID string, d Delegation,
) (ExternalAgentJob, error) {
	in, valid := canonicalExternalInput(in)
	if !valid || !sameExternalInput(d, in) {
		return ExternalAgentJob{}, domain.ErrBadRequest
	}
	if d.Status == DelegationDone || d.Status == DelegationFailed || d.Status == DelegationCancelled {
		return externalJob(jobID, d), nil
	}
	if !task.Status.Running() {
		return ExternalAgentJob{}, refusal("TASK_NOT_ACTIVE", "This task is stopped or finished. Start an active task before retrying work.")
	}
	roster := r.roster(ctx, task)
	if roster == nil {
		return ExternalAgentJob{}, refusal("ROLE_NOT_ALLOWED", "No enabled team is available for this task. Enable a team member in Settings, then try again.")
	}
	if err := s.externalParallelCheck(ctx, r, task, roster, in.AgentKey, jobID); err != nil {
		if errors.Is(err, domain.ErrBadRole) {
			return ExternalAgentJob{}, refusal("ROLE_NOT_ALLOWED", "That team role is not enabled. Choose an enabled role from list_agents.")
		}
		return ExternalAgentJob{}, err
	}
	if err := s.externalPlanRefusal(ctx, userID, task, r, roster, in.AgentKey); err != nil {
		return ExternalAgentJob{}, err
	}
	s.resumeExternal(r, task, jobID, d, roster)
	return externalJob(jobID, d), nil
}

func (s *Service) startExternal(
	ctx context.Context, userID string, task domain.Task, r *Runner, in ExternalAgentInput,
	jobID string,
) (ExternalAgentJob, error) {
	in, valid := canonicalExternalInput(in)
	if !valid {
		return ExternalAgentJob{}, domain.ErrBadRequest
	}
	roster := r.roster(ctx, task)
	if roster == nil {
		return ExternalAgentJob{}, refusal("ROLE_NOT_ALLOWED", "No enabled team is available for this task. Enable a team member in Settings, then try again.")
	}
	if err := s.externalParallelCheck(ctx, r, task, roster, in.AgentKey, jobID); err != nil {
		if errors.Is(err, domain.ErrBadRole) {
			return ExternalAgentJob{}, refusal("ROLE_NOT_ALLOWED", "That team role is not enabled. Choose an enabled role from list_agents.")
		}
		return ExternalAgentJob{}, err
	}
	if err := s.externalPlanRefusal(ctx, userID, task, r, roster, in.AgentKey); err != nil {
		return ExternalAgentJob{}, err
	}
	state := &runState{budget: &budget{left: r.lim.steps(true)}, segment: r.lim.delegateSteps(), roster: roster}
	state.clarifyAsked = maxClarifyHolds
	args := delegateArgs{Role: in.AgentKey, Brief: in.Brief, Context: in.Context, Effort: in.Effort, Scope: in.Scope}
	raw, err := json.Marshal(args)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	call := Call{ID: jobID, Name: toolDelegate, Input: raw}
	d, ok := r.startDelegation(ctx, task, state, call, args)
	if !ok {
		return ExternalAgentJob{}, refusal("POLICY_REFUSED", "The worker could not start under this task's current rules. Update the plan or required inputs, then try again.")
	}
	s.launchExternal(r, task, state, call, d)
	return ExternalAgentJob{ID: jobID, Role: d.RoleKey, Status: d.Status}, nil
}

func (s *Service) resumeExternal(r *Runner, task domain.Task, jobID string, d Delegation, roster *Roster) {
	if d.Status != DelegationWorking && d.Status != DelegationWaiting {
		return
	}
	if _, running := s.externalRunning[task.ID+"/"+jobID]; running {
		return
	}
	state := &runState{budget: &budget{left: r.lim.steps(true)}, segment: r.lim.delegateSteps(), roster: roster}
	state.clarifyAsked = maxClarifyHolds
	raw, _ := json.Marshal(delegateArgs{
		Role: d.RoleKey, Brief: d.Brief, Context: d.Context, Effort: d.RequestEffort, Scope: d.Scope,
	})
	s.launchExternal(r, task, state, Call{ID: jobID, Name: toolDelegate, Input: raw}, d)
}
