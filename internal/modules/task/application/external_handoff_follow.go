package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (s *Service) followUpHandoff(ctx context.Context, store HandoffStore, row PlanHandoff,
	external, linked domain.Task, prompt string,
) error {
	if !validHandoffLink(external, linked) {
		return domain.ErrWrongState
	}
	marker := "handoff:" + row.ID
	events, err := s.deps.Events.EventsAfter(ctx, linked.ID, 0)
	if err != nil {
		return fmt.Errorf("read linked task events: %w", err)
	}
	for _, event := range events {
		if event.Technical == marker {
			at := s.deps.Now()
			return followHandoff(ctx, store, row, linked, marker, prompt, at)
		}
	}
	if linked.Status.Running() {
		return domain.ErrBusy
	}
	if linked.Status == domain.StatusNeedsInput {
		return domain.ErrNeedsAnswer
	}
	at := s.deps.Now()
	return followHandoff(ctx, store, row, linked, marker, prompt, at)
}

func validHandoffLink(external, linked domain.Task) bool {
	return linked.UserID == external.UserID && linked.ProjectID == external.ProjectID &&
		linked.OriginTaskID == external.ID && linked.Source == "" && linked.WorkID == "" && linked.ArchivedAt == nil
}

func followHandoff(ctx context.Context, store HandoffStore, row PlanHandoff, linked domain.Task,
	marker, prompt string, at time.Time,
) error {
	if err := store.FollowHandoff(ctx, row, linked, handoffRequest(row, marker, at), prompt, at); err != nil {
		return fmt.Errorf("follow plan handoff: %w", err)
	}
	return nil
}

func handoffRequest(row PlanHandoff, marker string, at time.Time) domain.Event {
	return domain.Event{
		Kind: domain.EventRequest, Text: handoffMessage(row.Revision), Technical: marker, Time: at,
	}
}
