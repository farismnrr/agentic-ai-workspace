package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// HandoffStore owns durable claim, task creation, recovery, and retry state.
type HandoffStore interface {
	DueHandoffs(context.Context, time.Time, int) ([]string, error)
	ClaimHandoff(context.Context, string, time.Time) (PlanHandoff, bool, error)
	CreateHandoffTask(
		context.Context, PlanHandoff, domain.Task, domain.Event, string, time.Time,
	) (domain.Task, bool, error)
	FollowHandoff(context.Context, PlanHandoff, domain.Task, domain.Event, string, time.Time) error
	MarkHandoffDone(context.Context, string, time.Time) error
	FailHandoff(context.Context, string, string, time.Time) error
	ResetHandoffLeases(context.Context, time.Time) error
	RecoveringHandoff(context.Context, string) (bool, error)
}

// PlanHandoff combines the outbox row with the immutable approved snapshot.
type PlanHandoff struct {
	ID             string
	DecisionID     string
	WorkID         string
	TaskID         string
	UserID         string
	ProjectID      string
	Client         string
	State          string
	InternalTaskID string
	Revision       domain.PlanRevision
	Attempts       int
}
