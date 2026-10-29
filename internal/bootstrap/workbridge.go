package bootstrap

import (
	"context"
	"fmt"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const planFieldsPerStep = 3

// appWorks adapts the task use cases to the MCP adapter's consumer-owned port.
type appWorks struct{ works *taskapp.Works }

func (a appWorks) Validate(ctx context.Context, user, client, project, workID string) error {
	err := a.works.Validate(ctx, user, client, project, workID)
	return workBridgeError("validate work session", err)
}

func (a appWorks) AvailableRoles(ctx context.Context, user string) ([]codingmcp.WorkRole, error) {
	roles, err := a.works.AvailableRoles(ctx, user)
	if err != nil {
		return nil, workBridgeError("list work roles", err)
	}
	out := make([]codingmcp.WorkRole, 0, len(roles))
	for _, role := range roles {
		out = append(out, codingmcp.WorkRole{Key: role.Key, Name: role.Name, Purpose: role.Purpose})
	}
	return out, nil
}

func (a appWorks) Begin(ctx context.Context, user, client, project string,
	in codingmcp.BeginWorkInput,
) (codingmcp.BeginWorkResult, error) {
	result, err := a.works.Begin(ctx, taskapp.BeginInput{
		UserID: user, Client: client, ProjectID: project,
		Summary: in.Summary, Role: in.Role, IdempotencyKey: in.IdempotencyKey,
		NewTask: in.NewTask, PlannedRoles: in.PlannedRoles,
	})
	if err != nil {
		return codingmcp.BeginWorkResult{}, workBridgeError("begin work", err)
	}
	state, err := a.works.State(ctx, user, client, project, result.WorkID)
	if err != nil {
		return codingmcp.BeginWorkResult{}, workBridgeError("read begun work", err)
	}
	roles := make([]codingmcp.WorkRole, 0, len(result.Roles))
	for _, role := range result.Roles {
		roles = append(roles, codingmcp.WorkRole{Key: role.Key, Name: role.Name, Purpose: role.Purpose})
	}
	return codingmcp.BeginWorkResult{
		TaskID: result.TaskID, WorkID: result.WorkID, Roles: roles,
		Resumed: result.Resumed, Idempotency: result.Idempotency, Revision: state.Revision,
		State: string(state.State), Warnings: RedactionWarnings(in.Summary),
	}, nil
}

func (a appWorks) SetRole(
	ctx context.Context, user, client, project, workID, role, brief string,
) (codingmcp.RoleWorkResult, error) {
	before, err := a.works.State(ctx, user, client, project, workID)
	if err != nil {
		return codingmcp.RoleWorkResult{}, workBridgeError("read work role", err)
	}
	result, err := a.works.SetRoleWithBrief(ctx, user, client, project, workID, role, brief)
	if err != nil {
		return codingmcp.RoleWorkResult{}, workBridgeError("set work role", err)
	}
	state, err := a.works.State(ctx, user, client, project, workID)
	if err != nil {
		return codingmcp.RoleWorkResult{}, workBridgeError("read updated role", err)
	}
	return codingmcp.RoleWorkResult{
		WorkID: workID, Revision: state.Revision, State: string(state.State),
		Role:    codingmcp.WorkRole{Key: result.Key, Name: result.Name, Purpose: result.Purpose},
		Changed: before.Role != result.Key, Warnings: RedactionWarnings(role, brief),
	}, nil
}

func (a appWorks) UpdatePlan(ctx context.Context, user, client, project, workID string,
	in codingmcp.PlanWorkInput,
) (codingmcp.PlanWorkResult, error) {
	steps := make([]taskapp.PlanStepInput, 0, len(in.Steps))
	texts := make([]string, 0, len(in.Steps)*planFieldsPerStep)
	for _, step := range in.Steps {
		steps = append(steps, taskapp.PlanStepInput{
			ID: step.ID, Title: step.Title, Detail: step.Detail,
			State: taskdomain.PlanState(step.State),
		})
		texts = append(texts, step.ID, step.Title, step.Detail)
	}
	result, err := a.works.UpdatePlan(ctx, user, client, project, workID,
		taskapp.UpdatePlanInput{BaseRevision: in.BaseRevision, Steps: steps}, in.RequestID)
	if err != nil {
		return codingmcp.PlanWorkResult{}, workBridgeError("update work plan", err)
	}
	state, err := a.works.State(ctx, user, client, project, workID)
	if err != nil {
		return codingmcp.PlanWorkResult{}, workBridgeError("read updated plan", err)
	}
	return codingmcp.PlanWorkResult{
		WorkID: workID, Revision: state.Revision, State: string(state.State),
		PlanRevision: result.Revision, Steps: planSteps(result.Steps), Warnings: RedactionWarnings(texts...),
	}, nil
}

func (a appWorks) Complete(ctx context.Context, user, client, project, workID string,
	in codingmcp.CompleteWorkInput,
) (codingmcp.CompleteWorkResult, error) {
	result, err := a.works.Complete(ctx, user, client, project, workID, in.Outcome, in.Summary, in.Claims)
	if err != nil {
		return codingmcp.CompleteWorkResult{}, workBridgeError("complete work", err)
	}
	state, err := a.works.State(ctx, user, client, project, workID)
	if err != nil {
		return codingmcp.CompleteWorkResult{}, workBridgeError("read completed work", err)
	}
	return codingmcp.CompleteWorkResult{
		Already: result.Already, WorkID: workID, Revision: state.Revision, State: string(state.State),
		TaskStatus: completionStatus(in.Outcome), Verified: codingmcp.VerifiedWork{
			FilesChanged: result.Verified.FilesChanged,
			Runs:         result.Verified.Runs, LastRunPassed: result.Verified.LastRunPassed, Proven: result.Verified.Proven,
		},
		Warnings: RedactionWarnings(append([]string{in.Summary}, in.Claims...)...),
	}, nil
}

func (a appWorks) State(ctx context.Context, user, client, project, workID string) (codingmcp.WorkState, error) {
	result, err := a.works.State(ctx, user, client, project, workID)
	if err != nil {
		return codingmcp.WorkState{}, workBridgeError("read work state", err)
	}
	return stateResult(result), nil
}

func stateResult(in taskapp.StateResult) codingmcp.WorkState {
	var review *codingmcp.PlanReviewInfo
	if in.Review != nil {
		review = &codingmcp.PlanReviewInfo{
			Revision: in.Review.Revision, Summary: in.Review.Summary, Status: string(in.Review.Status),
			SmallTask: in.Review.SmallTask, Decision: in.Review.Decision, HandoffID: in.Review.HandoffID,
			HandoffState: in.Review.HandoffState, LinkedTaskID: in.Review.LinkedTaskID,
		}
	}
	return codingmcp.WorkState{
		TaskID: in.TaskID, WorkID: in.WorkID, Revision: in.Revision, State: string(in.State), Stale: in.Stale,
		Plan: planSteps(in.Plan), Progress: in.Progress, Role: in.Role, Research: in.Research, PRD: in.PRD,
		PlanState: string(in.PlanState), NextAction: in.NextAction, Review: review,
		Questions: stateQuestions(in.Questions),
		Evidence: codingmcp.EvidenceCounts{
			Claimed: in.Evidence.Claimed, Observed: in.Evidence.Observed, Ran: in.Evidence.Ran,
		},
		ResearchSize: in.ResearchSize, PRDSize: in.PRDSize, Warnings: []string{},
	}
}

func planSteps(in []taskdomain.PlanStep) []codingmcp.PlanWorkStep {
	out := make([]codingmcp.PlanWorkStep, 0, len(in))
	for _, step := range in {
		out = append(out, codingmcp.PlanWorkStep{
			ID: step.ID, Title: step.Title,
			Detail: step.Detail, State: string(step.State),
		})
	}
	return out
}

// RedactionWarnings reports whether secret-like text was removed from work documents.
func RedactionWarnings(texts ...string) []string {
	count := 0
	for _, text := range texts {
		_, found := taskdomain.Redact(text)
		count += found
	}
	if count == 0 {
		return []string{}
	}
	return []string{fmt.Sprintf("redacted %d possible secrets", count)}
}

func completionStatus(outcome string) string {
	switch outcome {
	case "completed":
		return string(taskdomain.StatusDone)
	case "failed":
		return string(taskdomain.StatusFailed)
	default:
		return string(taskdomain.StatusNeedsInput)
	}
}

func workBridgeError(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}
