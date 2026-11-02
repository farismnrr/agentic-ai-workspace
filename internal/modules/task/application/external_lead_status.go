package application

import "context"

// externalJobStatus reconciles interrupted external worker jobs on read. A
// persisted "working" record without a live worker is waiting for a safe retry,
// not falsely shown as still executing after a process restart.
func (s *Service) externalJobStatus(ctx context.Context, taskID string, d Delegation) (Delegation, error) {
	if d.Status != DelegationWorking {
		return d, nil
	}
	s.externalMu.Lock()
	_, running := s.externalRunning[taskID+"/"+d.ID]
	s.externalMu.Unlock()
	if running {
		return d, nil
	}
	if err := s.deps.Delegations.SetDelegationStatus(ctx, taskID, d.ID, DelegationWaiting); err != nil {
		return d, err
	}
	d.Status = DelegationWaiting
	return d, nil
}
