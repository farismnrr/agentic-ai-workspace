package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// ExternalAgentInput is a call from a separately authorized external lead.
// An idempotency key identifies one worker job inside an existing primary task.
type ExternalAgentInput struct {
	TaskID, AgentKey, Brief, Context, Effort, RequestKey string
	Scope                                                []string
}

// ExternalAgentJob is safe to send to a connected lead.
type ExternalAgentJob struct {
	ID     string `json:"job_id"`
	Role   string `json:"role"`
	Status string `json:"status"`
	Report string `json:"report,omitempty"`
}

func (s *Service) ExternalAgents(ctx context.Context, userID string) ([]AgentRole, error) {
	if s.deps.Team == nil {
		return nil, domain.ErrWrongState
	}
	roster, err := s.deps.Team.Roster(ctx, userID)
	if err != nil {
		return nil, err
	}
	return roster.Members, nil
}

func (s *Service) ExternalJob(ctx context.Context, userID, taskID, jobID string) (ExternalAgentJob, error) {
	if _, err := s.externalOwned(ctx, userID, taskID); err != nil {
		return ExternalAgentJob{}, err
	}
	d, found, err := s.deps.Delegations.FindDelegation(ctx, taskID, jobID)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	if !found {
		return ExternalAgentJob{}, domain.ErrNotFound
	}
	d, err = s.externalJobStatus(ctx, taskID, d)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	return ExternalAgentJob{ID: d.ID, Role: d.RoleKey, Status: d.Status, Report: d.Report}, nil
}

func (s *Service) externalOwned(ctx context.Context, userID, taskID string) (domain.Task, error) {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return task, err
	}
	if !strings.HasPrefix(task.Source, externalLeadSource) || task.ArchivedAt != nil {
		return domain.Task{}, domain.ErrWrongState
	}
	return task, nil
}

func (s *Service) externalActive(taskID string) bool {
	s.externalMu.Lock()
	defer s.externalMu.Unlock()
	for key := range s.externalRunning {
		if strings.HasPrefix(key, taskID+"/") {
			return true
		}
	}
	return false
}

// EnableExternalWorkers wires the SAME internal runner used by the normal scheduler.
func (s *Service) EnableExternalWorkers(runner *Runner) {
	s.externalMu.Lock()
	defer s.externalMu.Unlock()
	s.externalRunner = runner
	if s.externalRunning == nil {
		s.externalRunning = make(map[string]context.CancelFunc)
		s.externalCancelled = make(map[string]bool)
	}
}

// DelegateExternal starts a genuine internal teammate without ever starting the internal lead.
func (s *Service) DelegateExternal(ctx context.Context, userID string, in ExternalAgentInput) (ExternalAgentJob, error) {
	if in.AgentKey == "" || strings.TrimSpace(in.Brief) == "" || len(in.RequestKey) < 8 || len(in.RequestKey) > 128 {
		return ExternalAgentJob{}, domain.ErrBadRequest
	}
	task, err := s.externalOwned(ctx, userID, in.TaskID)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	if task.Status == domain.StatusDone || task.Status == domain.StatusFailed {
		return ExternalAgentJob{}, domain.ErrWrongState
	}
	sum := sha256.Sum256([]byte(task.ID + "\x00" + in.RequestKey))
	jobID := "external-" + hex.EncodeToString(sum[:16])
	s.externalMu.Lock()
	defer s.externalMu.Unlock()
	r := s.externalRunner
	if r == nil || r.deps.Delegations == nil {
		return ExternalAgentJob{}, domain.ErrWrongState
	}
	d, found, err := r.deps.Delegations.FindDelegation(ctx, task.ID, jobID)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	if found {
		if d.RoleKey != in.AgentKey || d.Brief != truncate(strings.TrimSpace(in.Brief), maxBrief) {
			return ExternalAgentJob{}, domain.ErrBadRequest
		}
		s.resumeExternal(r, task, in, jobID, d)
		return ExternalAgentJob{ID: d.ID, Role: d.RoleKey, Status: d.Status, Report: d.Report}, nil
	}
	roster := r.roster(ctx, task)
	if roster == nil {
		return ExternalAgentJob{}, domain.ErrWrongState
	}
	if err := s.externalParallelCheck(ctx, r, task, roster, in.AgentKey); err != nil {
		return ExternalAgentJob{}, err
	}
	state := &runState{budget: &budget{left: r.lim.steps(true)}, segment: r.lim.delegateSteps(), roster: roster}
	state.clarifyAsked = maxClarifyHolds
	args := delegateArgs{Role: in.AgentKey, Brief: in.Brief, Context: in.Context, Effort: in.Effort, Scope: cleanScope(in.Scope)}
	raw, err := json.Marshal(args)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	call := Call{ID: jobID, Name: toolDelegate, Input: raw}
	d, ok := r.startDelegation(ctx, task, state, call, args)
	if !ok {
		return ExternalAgentJob{}, fmt.Errorf("delegation refused by worker policy")
	}
	s.launchExternal(r, task, state, call, d)
	return ExternalAgentJob{ID: jobID, Role: d.RoleKey, Status: d.Status}, nil
}

// Jobs are detached from MCP request lifetime but bounded by the runner's wall-time.
func (s *Service) launchExternal(r *Runner, task domain.Task, state *runState, call Call, d Delegation) {
	key := task.ID + "/" + d.ID
	if _, running := s.externalRunning[key]; running {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), r.lim.wallTime(true))
	s.externalRunning[key] = cancel
	go func() {
		defer cancel()
		_, _ = r.runDelegation(ctx, task, state, call, d)
		s.externalMu.Lock()
		cancelled := s.externalCancelled[key]
		delete(s.externalRunning, key)
		delete(s.externalCancelled, key)
		s.externalMu.Unlock()
		if cancelled {
			_ = r.deps.Delegations.SetDelegationStatus(context.Background(), task.ID, d.ID, "cancelled")
		}
	}()
}

func (s *Service) resumeExternal(r *Runner, task domain.Task, in ExternalAgentInput, jobID string, d Delegation) {
	if d.Status != DelegationWorking && d.Status != DelegationWaiting {
		return
	}
	if _, running := s.externalRunning[task.ID+"/"+jobID]; running {
		return
	}
	roster := r.roster(context.Background(), task)
	if roster == nil {
		return
	}
	state := &runState{budget: &budget{left: r.lim.steps(true)}, segment: r.lim.delegateSteps(), roster: roster}
	state.clarifyAsked = maxClarifyHolds
	d.Scope = cleanScope(in.Scope)
	raw, _ := json.Marshal(delegateArgs{Role: in.AgentKey, Brief: in.Brief, Context: in.Context, Scope: d.Scope})
	s.launchExternal(r, task, state, Call{ID: jobID, Name: toolDelegate, Input: raw}, d)
}
