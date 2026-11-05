package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type observedEvents struct {
	EventStore
	notify Notifier
	owners *owners
}

func (o observedEvents) SubmitExternalLeadApproval(
	ctx context.Context, userID, taskID, source string, event domain.Event, at time.Time,
) (domain.Event, error) {
	submitter, ok := o.EventStore.(ExternalLeadPlanApprovalStore)
	if !ok {
		return domain.Event{}, domain.ErrWrongState
	}
	stored, err := submitter.SubmitExternalLeadApproval(ctx, userID, taskID, source, event, at)
	if err == nil {
		o.notify.Changed(userID, taskID)
	}
	return stored, err
}

func (o observedEvents) AnswerExternalLeadQuestion(
	ctx context.Context, userID, taskID, source string, event domain.Event, at time.Time,
) error {
	answerer, ok := o.EventStore.(ExternalLeadAnswerStore)
	if !ok {
		return domain.ErrWrongState
	}
	if err := answerer.AnswerExternalLeadQuestion(ctx, userID, taskID, source, event, at); err != nil {
		return err
	}
	o.notify.Changed(userID, taskID)
	return nil
}

// ApproveExternalPlanEvent forwards the Orchestrator-only atomic approval
// through the event observer without changing any Coding Space behavior.
func (o observedEvents) ApproveExternalPlanEvent(
	ctx context.Context, userID, client, taskID, eventID, requestHash string, at time.Time,
) error {
	approver, ok := o.EventStore.(externalPlanApprover)
	if !ok {
		return domain.ErrWrongState
	}
	if err := approver.ApproveExternalPlanEvent(ctx, userID, client, taskID, eventID, requestHash, at); err != nil {
		return err
	}
	o.notify.Changed(userID, taskID)
	return nil
}

func (o observedEvents) AppendEvent(ctx context.Context, taskID string, event domain.Event) (domain.Event, error) {
	stored, err := o.EventStore.AppendEvent(ctx, taskID, event)
	if err == nil {
		o.notify.Changed(o.owners.of(ctx, taskID), taskID)
	}

	return stored, err //nolint:wrapcheck // a transparent decorator
}

func (o observedEvents) UpdateEvent(ctx context.Context, taskID string, event domain.Event) error {
	err := o.EventStore.UpdateEvent(ctx, taskID, event)
	if err == nil {
		o.notify.Changed(o.owners.of(ctx, taskID), taskID)
	}

	return err //nolint:wrapcheck // a transparent decorator
}

func (o observedEvents) EventsChangedAfter(ctx context.Context, taskID string, version int64) ([]domain.Event, error) {
	events, err := o.EventStore.EventsChangedAfter(ctx, taskID, version)
	if err != nil {
		return nil, fmt.Errorf("load changed thread events: %w", err)
	}
	return events, nil
}
