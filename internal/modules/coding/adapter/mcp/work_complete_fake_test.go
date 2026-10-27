package mcp

import (
	"context"

	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (f *fakeWorkSessions) Complete(
	_ context.Context, _, _, _, workID string, in CompleteWorkInput,
) (CompleteWorkResult, error) {
	if in.Outcome == "completed" && (f.state.PlanState == "ready_for_approval" ||
		f.state.PlanState == "awaiting_clarification") {
		return CompleteWorkResult{}, taskdomain.ErrAwaitingPerson
	}
	f.state.State = "completed"
	return CompleteWorkResult{
		WorkID: workID, Revision: f.state.Revision + 1, State: "completed",
		TaskStatus: "done", Warnings: []string{},
	}, nil
}
