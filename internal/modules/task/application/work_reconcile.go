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
	cutoff := w.now().Add(-w.thresholds.IdleAfter)
	// Running commands can outlive the ordinary inactivity window, but an
	// orphaned command must not keep a task open forever.
	if w.sweepMode == "on" {
		hardCutoff := w.now().Add(-4 * w.thresholds.AbandonAfter)
		if err := w.interruptRunningCalls(ctx, &hardCutoff); err != nil {
			return err
		}
	}

	works, err := w.store.StaleOpenAll(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("list stale work for reconciliation: %w", err)
	}
	for _, work := range works {
		if waitingOnPerson(work.PlanState) || w.hasRunningCall(ctx, work) {
			continue
		}
		if w.now().Sub(work.UpdatedAt) < w.inactivityTimeout(ctx, work) {
			continue
		}
		if w.sweepMode == "dry-run" {
			w.logCandidate(work)
			continue
		}
		if err := w.autoCompleteInactive(ctx, work); err != nil {
			return err
		}
	}
	return nil
}

// RunReconciler sweeps inactive work and eventually interrupts orphaned calls.
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
	log.Printf("work sweep (dry-run): would auto-complete %s %s", work.ID, state)
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
	log.Printf("work sweep (dry-run): would auto-complete %s %s after idle window %s",
		work.ID, state, w.thresholds.IdleAfter)
}

// inactivityTimeout allows longer for sessions that have recorded actual work.
func (w *Works) inactivityTimeout(ctx context.Context, work domain.ExternalWork) time.Duration {
	if work.ReportedAt != nil {
		return w.thresholds.IdleAfter
	}
	events, err := w.deps.Events.EventsAfter(ctx, work.TaskID, 0)
	if err != nil {
		return w.thresholds.AbandonAfter // Fail conservatively if history is unavailable.
	}
	for _, event := range events {
		if event.Work != nil && event.Work.ID == work.ID &&
			(event.Kind == domain.EventAction || event.Kind == domain.EventResearch ||
				event.Kind == domain.EventPlan || event.Kind == domain.EventPRD ||
				event.Kind == domain.EventReceipt) {
			return w.thresholds.AbandonAfter
		}
	}
	return w.thresholds.IdleAfter
}

func (w *Works) autoCompleteInactive(ctx context.Context, work domain.ExternalWork) error {
	now := w.now()
	state, status := domain.WorkCompleted, domain.StatusDone
	text := "Auto-completed (inactive); unverified — no final report was received"
	if work.ReportedAt != nil {
		text = "Auto-completed after reported result; unverified"
	}
	proof := &domain.Proof{Proven: false}
	_, _, err := w.store.Commit(ctx, WorkChange{
		WorkID: work.ID, UserID: work.UserID, ExpectRev: work.Rev,
		Patch:  &WorkPatch{State: &state, EndedAt: &now},
		Task:   &TaskPatch{Status: &status, Summary: &text, Proof: proof},
		Events: []domain.Event{autoCompletionEvent(work, text, now)},
	})
	if errors.Is(err, domain.ErrStaleRevision) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("auto-complete inactive work: %w", err)
	}
	w.notify(ctx, domain.Task{ID: work.TaskID, UserID: work.UserID})
	return nil
}

func autoCompletionEvent(work domain.ExternalWork, text string, now time.Time) domain.Event {
	event := workEvent(work, domain.EventMessage, text, "", now)
	event.Work.Outcome = "auto_completed_unverified"
	return event
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
