package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// ExternalAgentInput is a call from a separately authorized external lead.
// An idempotency key identifies one worker job inside an existing primary task.
type ExternalAgentInput struct {
	TaskID, ClientID, AgentKey, Brief, Context, Effort, RequestKey string
	Scope                                                          []string
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

func (s *Service) externalActiveLocked(taskID string) bool {
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
	sum := sha256.Sum256([]byte(in.TaskID + "\x00" + in.RequestKey))
	jobID := "external-" + hex.EncodeToString(sum[:16])
	s.externalMu.Lock()
	defer s.externalMu.Unlock()
	task, err := s.externalOwnedByClient(ctx, userID, in.ClientID, in.TaskID)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	r := s.externalRunner
	if r == nil || r.deps.Delegations == nil {
		return ExternalAgentJob{}, domain.ErrWrongState
	}
	d, found, err := r.deps.Delegations.FindDelegation(ctx, task.ID, jobID)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	if found {
		return s.retryExternal(ctx, userID, task, r, in, jobID, d)
	}
	return s.startExternal(ctx, userID, task, r, in, jobID)
}

// Jobs are detached from MCP request lifetime but bounded by the runner's wall-time.
func (s *Service) launchExternal(r *Runner, task domain.Task, state *runState, call Call, d Delegation) {
	key := task.ID + "/" + d.ID
	if _, running := s.externalRunning[key]; running {
		return
	}
	workerRunner := externalWorkerRunner(r)
	ctx, cancel := context.WithTimeout(context.Background(), r.lim.wallTime(true))
	s.externalRunning[key] = cancel
	go func() {
		defer cancel()
		_, _ = workerRunner.runDelegation(ctx, task, state, call, d)
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
