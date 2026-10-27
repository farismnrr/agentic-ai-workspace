package application

import (
	"context"
)

// WaitContext waits for a running task until it finishes, ctx ends, or the
// scheduler wait limit expires. It returns false when it stops waiting early.
func (s *Scheduler) WaitContext(ctx context.Context, taskID string) bool {
	waitCtx, cancel := context.WithTimeout(ctx, schedulerWaitTimeout)
	defer cancel()
	stop := context.AfterFunc(waitCtx, func() {
		s.mu.Lock()
		s.cond.Broadcast()
		s.mu.Unlock()
	})
	defer stop()

	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		if _, running := s.running[taskID]; !running {
			return true
		}
		if err := waitCtx.Err(); err != nil {
			return false
		}
		s.cond.Wait()
	}
}
