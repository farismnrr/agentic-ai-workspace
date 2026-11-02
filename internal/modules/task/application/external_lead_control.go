package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const DelegationCancelled = "cancelled"

// ExternalJobs returns the durable worker jobs recorded inside one primary task.
func (s *Service) ExternalJobs(ctx context.Context, userID, taskID string) ([]ExternalAgentJob, error) {
	if _, err := s.externalOwned(ctx, userID, taskID); err != nil {
		return nil, err
	}
	reader, ok := s.deps.Delegations.(interface {
		ListDelegations(context.Context, string) ([]Delegation, error)
	})
	if !ok {
		return nil, domain.ErrWrongState
	}
	records, err := reader.ListDelegations(ctx, taskID)
	if err != nil {
		return nil, err
	}
	jobs := make([]ExternalAgentJob, 0, len(records))
	for _, d := range records {
		if len(d.ID) < len("external-") || d.ID[:len("external-")] != "external-" {
			continue
		}
		d, err = s.externalJobStatus(ctx, taskID, d)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, ExternalAgentJob{ID: d.ID, Role: d.RoleKey, Status: d.Status, Report: d.Report})
	}
	return jobs, nil
}

// CancelExternalJob only cancels an external-led worker. Internal lead delegation
// jobs are never reachable through this entrypoint.
func (s *Service) CancelExternalJob(ctx context.Context, userID, taskID, jobID string) (ExternalAgentJob, error) {
	if _, err := s.externalOwned(ctx, userID, taskID); err != nil {
		return ExternalAgentJob{}, err
	}
	if s.deps.Delegations == nil {
		return ExternalAgentJob{}, domain.ErrWrongState
	}
	d, found, err := s.deps.Delegations.FindDelegation(ctx, taskID, jobID)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	if !found || len(jobID) < len("external-") || jobID[:len("external-")] != "external-" {
		return ExternalAgentJob{}, domain.ErrNotFound
	}
	if d.Status == DelegationDone || d.Status == DelegationFailed || d.Status == DelegationCancelled {
		return ExternalAgentJob{ID: jobID, Role: d.RoleKey, Status: d.Status, Report: d.Report}, nil
	}
	key := taskID + "/" + jobID
	s.externalMu.Lock()
	cancel := s.externalRunning[key]
	if cancel != nil {
		s.externalCancelled[key] = true
	}
	s.externalMu.Unlock()
	if err := s.deps.Delegations.SetDelegationStatus(ctx, taskID, jobID, DelegationCancelled); err != nil {
		return ExternalAgentJob{}, err
	}
	if cancel != nil {
		cancel()
	}
	return ExternalAgentJob{ID: jobID, Role: d.RoleKey, Status: DelegationCancelled}, nil
}

// StopExternalLeadTask marks the external-led task done and cancels its active
// worker jobs. It cannot stop or alter a normal internal task.
func (s *Service) StopExternalLeadTask(ctx context.Context, userID, taskID string) error {
	task, err := s.externalOwned(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if task.Status == domain.StatusDone {
		return nil
	}
	s.externalMu.Lock()
	cancels := make([]context.CancelFunc, 0)
	for key, cancel := range s.externalRunning {
		if len(key) > len(taskID) && key[:len(taskID)+1] == taskID+"/" {
			s.externalCancelled[key] = true
			cancels = append(cancels, cancel)
		}
	}
	s.externalMu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	if err := s.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusDone, s.now()); err != nil {
		return err
	}
	_, err = s.deps.Events.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventMessage, Text: "Stopped by the external lead", Time: s.now(),
	})
	return err
}
