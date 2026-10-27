package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func submitStorePlan(t *testing.T, ctx context.Context, store *Store, work domain.ExternalWork) domain.PlanRevision {
	t.Helper()
	plan, err := store.SubmitPlan(ctx, application.PlanSubmission{
		WorkID: work.ID, UserID: work.UserID, ExpectRev: work.Rev,
		Revision: domain.PlanRevision{
			ID: "revision-" + work.ID, WorkID: work.ID, TaskID: work.TaskID, RevNo: 1,
			PlanRev: 1, Steps: []domain.PlanStep{{ID: "step-1", Title: "Build"}},
			PRD: "requirements snapshot", Research: "research snapshot", QA: `[{"answer":["yes"]}]`,
			Summary: "Approved outline", Digest: "snapshot-digest", Status: domain.PlanRevisionSubmitted,
		},
		Event: domain.Event{
			Kind: domain.EventPlanReview, Text: "Plan ready for approval",
			PlanReview: &domain.PlanReview{Revision: 1, Summary: "Approved outline", Status: domain.PlanRevisionSubmitted},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func approvalWrite(work domain.ExternalWork, revision int64, at time.Time) application.PlanDecisionWrite {
	return application.PlanDecisionWrite{
		Decision: domain.PlanDecision{
			ID: "decision-" + work.ID, WorkID: work.ID, TaskID: work.TaskID,
			UserID: work.UserID, ProjectID: work.ProjectID, Kind: domain.PlanDecisionApproved,
			Source: "mcp", Client: work.Client, IdemKey: "approve-key-123", ReqHash: "approval-hash",
			DecidedAt: at,
		}, Revision: revision, HandoffID: "handoff-" + work.ID,
	}
}
