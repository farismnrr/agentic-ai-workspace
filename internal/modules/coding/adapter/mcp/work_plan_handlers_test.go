package mcp

import "context"

func (f *fakeWorkSessions) SubmitPlan(
	_ context.Context, _, _, _, workID string, in SubmitPlanInput,
) (SubmitPlanResult, error) {
	f.state.PlanState = "ready_for_approval"
	f.state.Review = &PlanReviewInfo{Revision: in.BaseRevision, Summary: in.Summary, Status: "submitted"}
	f.state.Revision++
	return SubmitPlanResult{
		WorkID: workID, WorkRev: f.state.Revision, State: f.state.State, PlanState: f.state.PlanState,
		PlanRevision: in.BaseRevision, Summary: in.Summary, Status: "submitted", Warnings: []string{},
	}, nil
}
