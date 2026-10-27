package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// PlanDecisionStore atomically records a decision and, for approval, a handoff outbox row.
type PlanDecisionStore interface {
	DecidePlan(context.Context, PlanDecisionWrite) (PlanDecisionResult, error)
}

// HandoffKicker starts durable outbox delivery after approval commits.
type HandoffKicker interface {
	Kick(context.Context, string) (string, error)
}

// PlanDecisionWrite is the authenticated decision and transaction identifiers.
type PlanDecisionWrite struct {
	Decision  domain.PlanDecision
	HandoffID string
	Revision  int64
}

// PlanDecisionResult identifies the persisted approval and delivery state.
type PlanDecisionResult struct {
	Decision       domain.PlanDecision `json:"decision"`
	HandoffID      string              `json:"handoff_id,omitempty"`
	HandoffState   string              `json:"handoff_state,omitempty"`
	InternalTaskID string              `json:"internal_task_id,omitempty"`
}

// PlanReviewNotifier wakes the task thread after a review decision.
type PlanReviewNotifier interface {
	Changed(userID, taskID string)
}
