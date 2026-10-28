package bootstrap

import (
	"context"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (a appWorks) SubmitPlan(ctx context.Context, user, client, project, workID string,
	in codingmcp.SubmitPlanInput,
) (codingmcp.SubmitPlanResult, error) {
	result, err := a.works.SubmitPlan(ctx, user, client, project, workID, taskapp.SubmitPlanInput{
		BaseRevision: in.BaseRevision, Summary: in.Summary, RequestID: in.RequestID, SmallTask: in.SmallTask,
	})
	if err != nil {
		return codingmcp.SubmitPlanResult{}, workBridgeError("submit plan", err)
	}
	return codingmcp.SubmitPlanResult{
		WorkID: workID, WorkRev: result.WorkRev, State: string(result.WorkState),
		PlanState: string(taskdomain.PlanReadyForApproval), PlanRevision: result.Revision.RevNo,
		Summary: result.Revision.Summary, Status: string(result.Revision.Status),
		Warnings: RedactionWarnings(in.Summary),
	}, nil
}
