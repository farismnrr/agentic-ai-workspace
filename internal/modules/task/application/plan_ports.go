package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// PlanStore owns the transactional boundary for immutable submitted revisions.
type PlanStore interface {
	SubmitPlan(context.Context, PlanSubmission) (domain.PlanRevision, error)
	LatestPlan(context.Context, string) (domain.PlanRevision, bool, error)
	LatestPlanReview(context.Context, string) (domain.PlanReview, bool, error)
}

// PlanSnapshotReader loads immutable snapshots for the owner's review screen.
type PlanSnapshotReader interface {
	PlanRevision(context.Context, string, int64) (domain.PlanRevision, bool, error)
	LatestPlanReview(context.Context, string) (domain.PlanReview, bool, error)
}

// PlanSubmission contains the compare-and-swap context and immutable event snapshot.
type PlanSubmission struct {
	Revision  domain.PlanRevision
	WorkID    string
	UserID    string
	Event     domain.Event
	ExpectRev int64
}
