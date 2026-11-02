package application

import (
	"context"
	"log/slog"
	"time"

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
	trace := externalCancelTrace{started: time.Now(), outcome: "error"}
	defer trace.log()
	d, found, err := s.lookupExternalCancellation(ctx, userID, taskID, jobID, &trace)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	if !found || len(jobID) < len("external-") || jobID[:len("external-")] != "external-" {
		return ExternalAgentJob{}, domain.ErrNotFound
	}
	if d.Status == DelegationDone || d.Status == DelegationFailed || d.Status == DelegationCancelled {
		trace.outcome = d.Status
		return externalJob(jobID, d), nil
	}
	return s.cancelRunningExternalJob(ctx, taskID, jobID, &trace)
}

func (s *Service) lookupExternalCancellation(ctx context.Context, userID, taskID, jobID string, trace *externalCancelTrace) (Delegation, bool, error) {
	started := time.Now()
	defer func() { trace.lookup = time.Since(started) }()
	if _, err := s.externalOwned(ctx, userID, taskID); err != nil {
		return Delegation{}, false, err
	}
	if s.deps.Delegations == nil {
		return Delegation{}, false, domain.ErrWrongState
	}
	return s.deps.Delegations.FindDelegation(ctx, taskID, jobID)
}

func (s *Service) cancelRunningExternalJob(ctx context.Context, taskID, jobID string, trace *externalCancelTrace) (ExternalAgentJob, error) {
	key := taskID + "/" + jobID
	started := time.Now()
	s.externalMu.Lock()
	trace.mutexWait = time.Since(started)
	cancel := s.externalRunning[key]
	if cancel != nil {
		s.externalCancelled[key] = true
		// Stop the live worker before waiting for durable storage. This closes
		// the window where a slow status write leaves it running uncontrolled.
		started = time.Now()
		cancel()
		trace.contextCancel = time.Since(started)
	}
	s.externalMu.Unlock()
	started = time.Now()
	err := s.deps.Delegations.SetDelegationStatus(ctx, taskID, jobID, DelegationCancelled)
	trace.terminalWrite = time.Since(started)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	// The worker may have completed just before the cancellation write. Return
	// the persisted winner so a completed job is not reported as cancelled.
	started = time.Now()
	defer func() { trace.reread = time.Since(started) }()
	d, found, err := s.deps.Delegations.FindDelegation(ctx, taskID, jobID)
	if err != nil {
		return ExternalAgentJob{}, err
	}
	if !found {
		return ExternalAgentJob{}, domain.ErrNotFound
	}
	trace.outcome = d.Status
	return externalJob(jobID, d), nil
}

func externalJob(jobID string, d Delegation) ExternalAgentJob {
	return ExternalAgentJob{ID: jobID, Role: d.RoleKey, Status: d.Status, Report: d.Report}
}

type externalCancelTrace struct {
	started                                                 time.Time
	lookup, mutexWait, contextCancel, terminalWrite, reread time.Duration
	outcome                                                 string
}

func (trace *externalCancelTrace) log() {
	slog.Info("external delegation cancellation", "outcome", trace.outcome,
		"lookup_ms", trace.lookup.Milliseconds(), "mutex_wait_ms", trace.mutexWait.Milliseconds(),
		"context_cancel_ms", trace.contextCancel.Milliseconds(), "terminal_write_ms", trace.terminalWrite.Milliseconds(),
		"reread_ms", trace.reread.Milliseconds(), "total_ms", time.Since(trace.started).Milliseconds())
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
