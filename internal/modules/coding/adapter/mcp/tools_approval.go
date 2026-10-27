package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func (t *tools) addApprovalTools(server *sdk.Server) {
	sdk.AddTool(server, toolOf(domain.ToolApprovePlan), guarded(t.guard, domain.ToolApprovePlan,
		func(approvePlanInput) string { return "approve an external plan" }, t.approvePlan))
}

func (t *tools) approvePlan(ctx context.Context, in approvePlanInput) (PlanApprovalResult, report, error) {
	who, projectID, err := approvalWorkContext(ctx, t, in.WorkTarget)
	if err != nil {
		return PlanApprovalResult{}, report{}, err
	}
	result, err := t.works.ApprovePlan(ctx, who.User, who.Client, projectID, in.WorkID,
		in.Revision, in.IdempotencyKey)
	if err != nil {
		return PlanApprovalResult{}, report{}, workToolError("approve plan", err)
	}
	return PlanApprovalResult{
		Decision: result.Decision, HandoffID: result.HandoffID,
		HandoffState: result.HandoffState, InternalTaskID: result.InternalTaskID,
		Revision: in.Revision,
	}, report{}, nil
}

func approvalWorkContext(ctx context.Context, t *tools, target WorkTarget) (identity, string, error) {
	who, _ := ctx.Value(identityKey{}).(identity)
	_, projectID, err := t.approvalWorkServices(ctx, target.Project)
	return who, projectID, err
}
