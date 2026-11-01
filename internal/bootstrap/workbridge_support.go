package bootstrap

import (
	"fmt"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

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
		TaskID: in.TaskID, WorkID: in.WorkID, Revision: in.Revision, PlanRevision: in.PlanRevision,
		State: string(in.State), Stale: in.Stale,
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
