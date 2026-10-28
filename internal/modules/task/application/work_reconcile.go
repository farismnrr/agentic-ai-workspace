package application

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const defaultWorkSweepEvery = time.Minute

// UseSweepMode configures whether reconciliation only reports candidates or closes stale work.
func (w *Works) UseSweepMode(mode string) {
	if mode == "on" || mode == "off" || mode == "dry-run" {
		w.sweepMode = mode
	}
}

// Reconcile inspects old open work and applies the configured sweep behavior.
func (w *Works) Reconcile(ctx context.Context) error {
	if w.sweepMode == "off" {
		return nil
	}
	cutoff := w.now().Add(-w.thresholds.AbandonAfter)
	if w.sweepMode == "on" {
		if err := w.interruptRunningCalls(ctx, &cutoff); err != nil {
			return err
		}
	}
	works, err := w.store.StaleOpenAll(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("list stale work for reconciliation: %w", err)
	}
	for _, work := range works {
		if waitingOnPerson(work.PlanState) {
			continue
		}
		if w.sweepMode == "dry-run" {
			w.logCandidate(work)
			continue
		}
		if err := w.abandon(ctx, work); err != nil {
			return err
		}
	}
	return nil
}

// RunReconciler sweeps stale work; running calls newer than the abandon window remain open.
func (w *Works) RunReconciler(ctx context.Context, every time.Duration) {
	if w.sweepMode == "off" {
		return
	}
	if every <= 0 {
		every = defaultWorkSweepEvery
	}
	w.runSweep(ctx)
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runSweep(ctx)
		}
	}
}

func (w *Works) runSweep(ctx context.Context) {
	if err := w.Reconcile(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("work sweep failed: %v", err)
	}
}

func (w *Works) logCandidate(work domain.ExternalWork) {
	w.dryRunMu.Lock()
	defer w.dryRunMu.Unlock()
	if _, logged := w.dryRunSeen[work.ID]; logged {
		return
	}
	w.dryRunSeen[work.ID] = struct{}{}
	state := "unreported"
	if work.ReportedAt != nil {
		state = "reported"
	}
	log.Printf("work sweep (dry-run): would abandon %s %s", work.ID, state)
}

func (w *Works) logCapCandidate(work domain.ExternalWork) {
	w.dryRunMu.Lock()
	defer w.dryRunMu.Unlock()
	key := work.ID + ":idle-cap"
	if _, logged := w.dryRunSeen[key]; logged {
		return
	}
	w.dryRunSeen[key] = struct{}{}
	state := "unreported"
	if work.ReportedAt != nil {
		state = "reported"
	}
	log.Printf("work sweep (dry-run): would abandon %s %s after idle window %s",
		work.ID, state, w.thresholds.IdleAfter)
}

func (w *Works) abandon(ctx context.Context, work domain.ExternalWork) error {
	now := w.now()
	state, ended := domain.WorkAbandoned, now
	status, text, kind := abandonmentResult(work.ReportedAt != nil)
	_, _, err := w.store.Commit(ctx, WorkChange{
		WorkID: work.ID, UserID: work.UserID, ExpectRev: work.Rev,
		Patch:  &WorkPatch{State: &state, EndedAt: &ended},
		Task:   &TaskPatch{Status: statusPtr(status)},
		Events: []domain.Event{abandonmentEvent(work.ID, now, text, kind)},
	})
	if errors.Is(err, domain.ErrStaleRevision) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("abandon stale work: %w", err)
	}
	w.notify(ctx, domain.Task{ID: work.TaskID, UserID: work.UserID})
	return nil
}

func abandonmentResult(reported bool) (domain.Status, string, domain.EventKind) {
	if reported {
		return domain.StatusReview, "Ready for your review", domain.EventMessage
	}
	return domain.StatusFailed, "Lost contact", domain.EventError
}

func abandonmentEvent(workID string, at time.Time, text string, kind domain.EventKind) domain.Event {
	return domain.Event{
		Kind: kind, Text: text, Time: at,
		Work: &domain.WorkRef{ID: workID, Outcome: string(domain.WorkAbandoned)},
	}
}

func (w *Works) interruptRunningCalls(ctx context.Context, before *time.Time) error {
	works, err := w.store.InterruptRunningCalls(ctx, before)
	if err != nil {
		return fmt.Errorf("interrupt orphaned work calls: %w", err)
	}
	for _, work := range works {
		w.notify(ctx, domain.Task{ID: work.TaskID, UserID: work.UserID})
	}
	return nil
}
