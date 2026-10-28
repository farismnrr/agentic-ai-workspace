package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const confirmedReceipt = "Marked complete by you after the assistant reported its result"

// Confirm closes reported or idle work with a receipt, using the loaded revision as a compare-and-swap.
func (w *Works) Confirm(ctx context.Context, userID, client, projectID, workID string) (Completion, error) {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return Completion{}, err
	}
	work, task, err := w.loadWorkTask(ctx, userID, client, projectID, workID)
	if err != nil {
		return Completion{}, err
	}
	if work.State == domain.WorkCompleted && hasConfirmationReceipt(ctx, w.deps.Events, task.ID, work.ID) {
		result := completionResult("completed", task.Proof)
		result.Already = true
		return result, nil
	}
	phase := domain.DeriveWorkPhase(work, w.now(), w.thresholds)
	if phase != domain.WorkPhaseReported && phase != domain.WorkPhaseIdle {
		return Completion{}, domain.ErrWrongState
	}
	result, err := w.confirmWork(ctx, work, task)
	if errors.Is(err, domain.ErrStaleRevision) && hasConfirmationReceipt(ctx, w.deps.Events, task.ID, work.ID) {
		result = completionResult("completed", nil)
		result.Verified.Proven = false
		result.Already = true
		return result, nil
	}
	return result, err
}

func (w *Works) confirmWork(ctx context.Context, work domain.ExternalWork, task domain.Task) (Completion, error) {
	events, err := w.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return Completion{}, fmt.Errorf("read work evidence: %w", err)
	}
	proof := buildProof(events, nil)
	proof.Proven = false
	state, status, ended := domain.WorkCompleted, domain.StatusDone, w.now()
	event := workEvent(work, domain.EventReceipt, confirmedReceipt, "", ended)
	event.Work.Outcome = string(state)
	summary := confirmedReceipt
	patch := &TaskPatch{Status: &status, Summary: &summary, Proof: proof}
	workPatch := &WorkPatch{State: &state, EndedAt: &ended}
	if err := w.appendMutation(ctx, work, patch, workPatch, []domain.Event{event}); err != nil {
		return Completion{}, err
	}
	return completionResult(string(state), proof), nil
}

func hasConfirmationReceipt(ctx context.Context, reader EventStore, taskID, workID string) bool {
	events, err := reader.EventsAfter(ctx, taskID, 0)
	if err != nil {
		return false
	}
	for _, event := range events {
		if event.Kind == domain.EventReceipt && event.Text == confirmedReceipt &&
			event.Work != nil && event.Work.ID == workID {
			return true
		}
	}
	return false
}
