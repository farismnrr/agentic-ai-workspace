package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func (t *tools) addWorkTools(server *sdk.Server) {
	sdk.AddTool(server, toolOf(domain.ToolBeginWork), guarded(t.guard, domain.ToolBeginWork,
		func(in beginWorkInput) string { return "begin work: " + in.Summary }, t.beginWork))
	sdk.AddTool(server, toolOf(domain.ToolSetRole), guarded(t.guard, domain.ToolSetRole,
		func(in roleWorkInput) string { return "set role: " + in.Role }, t.setRole))
	sdk.AddTool(server, toolOf(domain.ToolUpdatePlan), guarded(t.guard, domain.ToolUpdatePlan,
		func(workPlanInput) string { return "update work plan" }, t.updatePlan))
	sdk.AddTool(server, toolOf(domain.ToolSaveResearch), guarded(t.guard, domain.ToolSaveResearch,
		func(researchInput) string { return "save work research" }, t.saveResearch))
	sdk.AddTool(server, toolOf(domain.ToolSaveRequirements), guarded(t.guard, domain.ToolSaveRequirements,
		func(markdownInput) string { return "save work requirements" }, t.saveRequirements))
	sdk.AddTool(server, toolOf(domain.ToolReportProgress), guarded(t.guard, domain.ToolReportProgress,
		func(in progressInput) string { return "report work progress: " + in.Message }, t.reportProgress))
	sdk.AddTool(server, toolOf(domain.ToolCompleteWork), guarded(t.guard, domain.ToolCompleteWork,
		func(in completeInput) string { return "complete work: " + in.Outcome }, t.completeWork))
	sdk.AddTool(server, toolOf(domain.ToolGetWorkState), guarded(t.guard, domain.ToolGetWorkState,
		func(workIDInput) string { return "get work state" }, t.getWorkState))
}

func (t *tools) beginWork(ctx context.Context, in beginWorkInput) (BeginWorkResult, report, error) {
	who, _ := ctx.Value(identityKey{}).(identity)
	svc, projectID, err := t.workServices(ctx, in.Project, "")
	if err != nil {
		return BeginWorkResult{}, report{}, err
	}
	result, err := svc.Begin(ctx, who.User, who.Client, projectID, BeginWorkInput{
		Summary: in.Summary, Role: in.Role, IdempotencyKey: in.IdempotencyKey,
		NewTask: in.NewTask, PlannedRoles: in.PlannedRoles,
	})
	return result, report{}, workToolError("begin work", err)
}

func (t *tools) setRole(ctx context.Context, in roleWorkInput) (RoleWorkResult, report, error) {
	who, _ := ctx.Value(identityKey{}).(identity)
	// A role may have been disabled; changing to an enabled role must stay possible.
	_, projectID, err := t.approvalWorkServices(ctx, in.Project)
	if err != nil {
		return RoleWorkResult{}, report{}, err
	}
	result, err := t.works.SetRole(ctx, who.User, who.Client, projectID, in.WorkID, strings.TrimSpace(in.Role), in.Brief)
	return result, report{}, workToolError("set work role", err)
}

func (t *tools) updatePlan(ctx context.Context, in workPlanInput) (PlanWorkResult, report, error) {
	result, err := callWork(ctx, t, in.WorkTarget, "update work plan",
		func(who identity, projectID string) (PlanWorkResult, error) {
			return t.works.UpdatePlan(ctx, who.User, who.Client, projectID, in.WorkID,
				PlanWorkInput{BaseRevision: in.BaseRevision, RequestID: in.RequestID, Steps: in.Steps})
		})
	return result, report{}, workToolError("update work plan", err)
}

func (t *tools) saveResearch(ctx context.Context, in researchInput) (saveOutput, report, error) {
	who, projectID, err := t.workContext(ctx, in.WorkTarget)
	if err != nil {
		return saveOutput{}, report{}, err
	}
	result, err := t.works.SaveResearch(ctx, who.User, who.Client, projectID, in.WorkID,
		ResearchWorkInput{Markdown: in.Markdown, Sources: in.Sources})
	return savedOutput(result, "research.md", workToolError("save work research", err))
}

func (t *tools) saveRequirements(ctx context.Context, in markdownInput) (saveOutput, report, error) {
	who, projectID, err := t.workContext(ctx, in.WorkTarget)
	if err != nil {
		return saveOutput{}, report{}, err
	}
	result, err := t.works.SaveRequirements(ctx, who.User, who.Client, projectID, in.WorkID, in.Markdown)
	return savedOutput(result, "requirements.md", workToolError("save work requirements", err))
}

func (t *tools) reportProgress(ctx context.Context, in progressInput) (WorkMutation, report, error) {
	who, projectID, err := t.workContext(ctx, in.WorkTarget)
	if err != nil {
		return WorkMutation{}, report{}, err
	}
	result, err := t.works.Progress(ctx, who.User, who.Client, projectID, in.WorkID,
		ProgressWorkInput{Message: in.Message, Claims: in.Claims})
	return result, report{}, workToolError("report work progress", err)
}

func (t *tools) completeWork(ctx context.Context, in completeInput) (CompleteWorkResult, report, error) {
	who, _ := ctx.Value(identityKey{}).(identity)
	_, projectID, err := t.workServices(ctx, in.Project, "")
	if err != nil {
		return CompleteWorkResult{}, report{}, err
	}
	result, err := t.works.Complete(ctx, who.User, who.Client, projectID, in.WorkID,
		CompleteWorkInput{Outcome: in.Outcome, Summary: in.Summary, Claims: in.Claims})
	return result, report{}, workToolError("complete work", err)
}

func (t *tools) getWorkState(ctx context.Context, in workIDInput) (WorkState, report, error) {
	who, _ := ctx.Value(identityKey{}).(identity)
	_, projectID, err := t.approvalWorkServices(ctx, in.Project)
	if err != nil {
		return WorkState{}, report{}, err
	}
	result, err := t.works.State(ctx, who.User, who.Client, projectID, in.WorkID)
	return result, report{}, workToolError("read work state", err)
}

func (t *tools) workServices(ctx context.Context, project, workID string) (WorkSessions, string, error) {
	return t.resolveWorkServices(ctx, project, workID, true)
}

func (t *tools) approvalWorkServices(ctx context.Context, project string) (WorkSessions, string, error) {
	return t.resolveWorkServices(ctx, project, "", false)
}

func (t *tools) resolveWorkServices(ctx context.Context, project, workID string, validate bool) (
	WorkSessions, string, error,
) {
	if t.works == nil {
		return nil, "", fmt.Errorf("work sessions are unavailable: %w", errInternal)
	}
	who, _ := ctx.Value(identityKey{}).(identity)
	services, err := t.workspaces.For(ctx, who.User, who.Client, project)
	if err != nil {
		return nil, "", fmt.Errorf("open workspace: %w", err)
	}
	if validate && workID != "" {
		if err := t.works.Validate(ctx, who.User, who.Client, services.ProjectID, workID); err != nil {
			return nil, "", workToolError("validate work session", err)
		}
	}
	if state, ok := ctx.Value(stateKey{}).(*callState); ok {
		state.project = services.ProjectID
		if state.begin != nil && !state.started {
			state.started = true
			state.begin(services.ProjectID)
		}
	}
	return t.works, services.ProjectID, nil
}

func callWork[Output any](ctx context.Context, t *tools, target WorkTarget, action string,
	call func(identity, string) (Output, error),
) (Output, error) {
	who, projectID, err := t.workContext(ctx, target)
	if err != nil {
		var empty Output
		return empty, err
	}
	result, err := call(who, projectID)
	return result, workToolError(action, err)
}

func workToolError(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}

func (t *tools) workContext(ctx context.Context, target WorkTarget) (identity, string, error) {
	who, _ := ctx.Value(identityKey{}).(identity)
	_, projectID, err := t.workServices(ctx, target.Project, target.WorkID)
	return who, projectID, err
}
