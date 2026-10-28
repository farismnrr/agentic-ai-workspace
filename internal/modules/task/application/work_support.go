package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (w *Works) checkCaps(ctx context.Context, userID, projectID string) error {
	userCount, projectCount, err := w.countCaps(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if userCount < domain.MaxOpenWorksUser && projectCount < domain.MaxOpenWorksProject {
		return nil
	}
	err = w.abandonIdleAtCap(ctx, userID)
	if err != nil {
		return err
	}
	userCount, projectCount, err = w.countCaps(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if userCount >= domain.MaxOpenWorksUser || projectCount >= domain.MaxOpenWorksProject {
		return domain.ErrTooMany
	}
	return nil
}

func (w *Works) countCaps(ctx context.Context, userID, projectID string) (int, int, error) {
	userCount, err := w.store.CountOpen(ctx, userID, "")
	if err != nil {
		return 0, 0, fmt.Errorf("count user's open work: %w", err)
	}
	projectCount, err := w.store.CountOpen(ctx, userID, projectID)
	if err != nil {
		return 0, 0, fmt.Errorf("count project's open work: %w", err)
	}
	return userCount, projectCount, nil
}

func (w *Works) abandonIdleAtCap(ctx context.Context, userID string) error {
	if w.sweepMode == "off" {
		return nil
	}
	cutoff := w.now().Add(-w.thresholds.IdleAfter)
	works, err := w.store.StaleOpen(ctx, userID, cutoff)
	if err != nil {
		return fmt.Errorf("find idle work at project cap: %w", err)
	}
	for _, work := range works {
		if err := w.abandonAtCap(ctx, userID, work); err != nil {
			return err
		}
	}
	return nil
}

func (w *Works) abandonAtCap(ctx context.Context, userID string, work domain.ExternalWork) error {
	if waitingOnPerson(work.PlanState) || w.hasRunningCall(ctx, work) {
		return nil
	}
	if w.sweepMode == "dry-run" {
		w.logCapCandidate(work)
		return nil
	}
	now := w.now()
	state := domain.WorkAbandoned
	status, text, kind := abandonmentResult(work.ReportedAt != nil)
	_, _, err := w.store.Commit(ctx, WorkChange{
		WorkID: work.ID, UserID: userID, ExpectRev: work.Rev,
		Patch:  &WorkPatch{State: &state, EndedAt: &now},
		Task:   &TaskPatch{Status: statusPtr(status)},
		Events: []domain.Event{abandonmentEvent(work.ID, now, text, kind)},
	})
	if err != nil && !errors.Is(err, domain.ErrStaleRevision) {
		return fmt.Errorf("abandon idle work at project cap: %w", err)
	}
	if err == nil {
		w.notify(ctx, domain.Task{ID: work.TaskID, UserID: userID})
	}
	return nil
}

func (w *Works) hasRunningCall(ctx context.Context, work domain.ExternalWork) bool {
	events, err := w.deps.Events.EventsAfter(ctx, work.TaskID, 0)
	if err != nil {
		return true
	}
	for _, event := range events {
		if event.Work != nil && event.Work.ID == work.ID && event.Work.Phase == "running" {
			return true
		}
	}
	return false
}

func waitingOnPerson(state domain.PlanPhase) bool {
	return state == domain.PlanReadyForApproval || state == domain.PlanAwaitingClarification
}

func statusPtr(status domain.Status) *domain.Status { return &status }

func (w *Works) notify(_ context.Context, task domain.Task) {
	if w.deps.Notify != nil {
		w.deps.Notify.Changed(task.UserID, task.ID)
	}
}
