package bootstrap

import (
	"context"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
)

func (a appWorks) ApprovePlan(ctx context.Context, user, client, project, workID string,
	revision int64, idempotencyKey string,
) (codingmcp.PlanApprovalResult, error) {
	result, err := a.works.ApproveFromMCP(ctx, user, client, project, workID, revision, idempotencyKey)
	if err != nil {
		return codingmcp.PlanApprovalResult{}, workBridgeError("approve external plan", err)
	}
	return codingmcp.PlanApprovalResult{
		Decision: string(result.Decision.Kind), HandoffID: result.HandoffID,
		HandoffState: result.HandoffState, InternalTaskID: result.InternalTaskID,
		Revision: revision,
	}, nil
}
