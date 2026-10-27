package application

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	cutoff := w.now().Add(-30 * time.Minute)
	works, err := w.store.StaleOpen(ctx, userID, cutoff)
	if err != nil {
		return fmt.Errorf("find idle work at project cap: %w", err)
	}
	for _, work := range works {
		if w.hasRunningCall(ctx, work) {
			continue
		}
		state, ended := domain.WorkAbandoned, w.now()
		_, _, err := w.store.Commit(ctx, WorkChange{
			WorkID: work.ID, UserID: userID, ExpectRev: work.Rev,
			Patch:  &WorkPatch{State: &state, EndedAt: &ended},
			Task:   &TaskPatch{Status: statusPtr(domain.StatusFailed)},
			Events: []domain.Event{abandonedEvent(work.ID, ended)},
		})
		if err != nil && !errors.Is(err, domain.ErrStaleRevision) {
			return fmt.Errorf("abandon idle work at project cap: %w", err)
		}
		if err == nil {
			w.notify(ctx, domain.Task{ID: work.TaskID, UserID: userID})
		}
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

func (w *Works) abandonStale(ctx context.Context, userID string) error {
	cutoff := w.now().Add(-24 * time.Hour)
	works, err := w.store.StaleOpen(ctx, userID, cutoff)
	if err != nil {
		return fmt.Errorf("find abandoned work: %w", err)
	}
	for _, work := range works {
		state, ended := domain.WorkAbandoned, w.now()
		_, _, err := w.store.Commit(ctx, WorkChange{
			WorkID: work.ID, UserID: userID, ExpectRev: work.Rev,
			Patch:  &WorkPatch{State: &state, EndedAt: &ended},
			Task:   &TaskPatch{Status: statusPtr(domain.StatusFailed)},
			Events: []domain.Event{abandonedEvent(work.ID, ended)},
		})
		if errors.Is(err, domain.ErrStaleRevision) {
			continue
		}
		if err != nil {
			return fmt.Errorf("abandon stale work: %w", err)
		}
		w.notify(ctx, domain.Task{ID: work.TaskID, UserID: userID})
	}
	return nil
}

func abandonedEvent(workID string, at time.Time) domain.Event {
	return domain.Event{
		Kind: domain.EventError, Text: "Work was abandoned after 24 hours without activity.", Time: at,
		Work: &domain.WorkRef{ID: workID, Outcome: string(domain.WorkAbandoned)},
	}
}

func statusPtr(status domain.Status) *domain.Status { return &status }

func (w *Works) notify(_ context.Context, task domain.Task) {
	if w.deps.Notify != nil {
		w.deps.Notify.Changed(task.UserID, task.ID)
	}
}
