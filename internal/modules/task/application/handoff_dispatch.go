package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	handoffLease  = time.Minute
	handoffBatch  = 100
	handoffSettle = 30 * time.Second
)

// Handoffs drains the durable approval outbox into the internal task scheduler.
type Handoffs struct {
	service *Service
	store   HandoffStore
}

// NewHandoffs connects outbox persistence to the existing Orchestrator service.
func NewHandoffs(service *Service, store HandoffStore) *Handoffs {
	return &Handoffs{service: service, store: store}
}

// Kick attempts one approved plan immediately; retries remain recoverable in SQLite.
func (h *Handoffs) Kick(ctx context.Context, id string) (string, error) {
	row, claimed, err := h.store.ClaimHandoff(ctx, id, h.service.deps.Now())
	if err != nil {
		return "", fmt.Errorf("claim plan handoff: %w", err)
	}
	if !claimed {
		return "", nil
	}
	if row.State == "done" {
		return row.InternalTaskID, nil
	}
	var taskID string
	if row.State == "created" {
		taskID = row.InternalTaskID
	} else {
		taskID, err = h.dispatch(ctx, row)
	}
	if err != nil {
		_ = h.store.FailHandoff(context.WithoutCancel(ctx), row.ID, handoffErrorCategory(err), h.service.deps.Now())
		return "", fmt.Errorf("dispatch plan handoff: %w", err)
	}
	if err := h.finish(ctx, row, taskID); err != nil {
		return taskID, err
	}
	return taskID, nil
}

func (h *Handoffs) dispatch(ctx context.Context, row PlanHandoff) (string, error) {
	external, err := h.service.deps.Tasks.Find(ctx, row.UserID, row.TaskID)
	if err != nil || external.ProjectID != row.ProjectID || external.WorkID != row.WorkID ||
		external.Source != row.Client || external.ArchivedAt != nil {
		return "", domain.ErrWrongState
	}
	message := handoffMessage(row.Revision)
	prompt := handoffContext(row.Revision)
	task, event, err := h.service.buildLinkedTask(ctx, external, message)
	if err != nil {
		return "", err
	}
	event.Technical = "handoff:" + row.ID
	linked, created, err := h.store.CreateHandoffTask(ctx, row, task, event, prompt, h.service.deps.Now())
	if err != nil {
		return "", fmt.Errorf("create Orchestrator handoff task: %w", err)
	}
	if !created {
		if err := h.service.followUpHandoff(ctx, h.store, row, external, linked, prompt); err != nil {
			return "", err
		}
	}
	return linked.ID, nil
}

func (h *Handoffs) finish(ctx context.Context, row PlanHandoff, taskID string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("finish plan handoff: %w", err)
	}
	settle, cancel := context.WithTimeout(context.WithoutCancel(ctx), handoffSettle)
	defer cancel()
	if h.service.deps.Notify != nil {
		h.service.deps.Notify.Changed(row.UserID, row.TaskID)
		h.service.deps.Notify.Changed(row.UserID, taskID)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("finish plan handoff: %w", err)
	}
	if handoffQueue, ok := h.service.dispatcher.(interface {
		EnqueueHandoff(string, string)
	}); ok {
		handoffQueue.EnqueueHandoff(taskID, row.ProjectID)
	} else {
		h.service.dispatcher.Enqueue(taskID, row.ProjectID)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("finish plan handoff: %w", err)
	}
	if err := h.store.MarkHandoffDone(settle, row.ID, h.service.deps.Now()); err != nil {
		return fmt.Errorf("mark plan handoff delivered: %w", err)
	}
	return nil
}

// Sweep retries due pending handoffs and resumes committed task creations.
func (h *Handoffs) Sweep(ctx context.Context) error {
	ids, err := h.store.DueHandoffs(ctx, h.service.deps.Now(), handoffBatch)
	if err != nil {
		return fmt.Errorf("list due plan handoffs: %w", err)
	}
	var failures []error
	for _, id := range ids {
		if _, kickErr := h.Kick(ctx, id); kickErr != nil {
			failures = append(failures, kickErr)
		}
	}
	return errors.Join(failures...)
}

// Run performs an immediate recovery sweep and then checks the outbox periodically.
func (h *Handoffs) Run(ctx context.Context, every time.Duration) {
	if every <= 0 {
		every = time.Minute
	}
	_ = h.Sweep(ctx)
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = h.Sweep(ctx)
		}
	}
}

// PrepareRecovery clears leases left by a prior process before normal task recovery.
func (h *Handoffs) PrepareRecovery(ctx context.Context) error {
	if err := h.store.ResetHandoffLeases(ctx, h.service.deps.Now()); err != nil {
		return fmt.Errorf("reset plan handoff leases: %w", err)
	}
	return nil
}

// RecoveringTask protects a committed handoff task until the outbox re-enqueues it.
func (h *Handoffs) RecoveringTask(ctx context.Context, taskID string) (bool, error) {
	recovering, err := h.store.RecoveringHandoff(ctx, taskID)
	if err != nil {
		return false, fmt.Errorf("check task handoff recovery: %w", err)
	}
	return recovering, nil
}

func handoffErrorCategory(err error) string {
	switch {
	case errors.Is(err, domain.ErrBusy), errors.Is(err, domain.ErrTryAgain):
		return "busy"
	case errors.Is(err, domain.ErrNoModel):
		return "model_unavailable"
	default:
		return "dispatch_failed"
	}
}

var _ HandoffKicker = (*Handoffs)(nil)
