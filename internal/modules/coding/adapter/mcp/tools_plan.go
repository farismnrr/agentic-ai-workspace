package mcp

import (
	"context"
	"encoding/json"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func (t *tools) addPlanTools(server *sdk.Server) {
	sdk.AddTool(server, toolOf(domain.ToolSubmitPlan), guarded(t.guard, domain.ToolSubmitPlan,
		func(submitPlanInput) string { return "submit external plan for review" }, t.submitPlan))
	sdk.AddTool(server, toolOf(domain.ToolFinishResearch), guarded(t.guard, domain.ToolFinishResearch,
		func(finishResearchInput) string { return "check research readiness" }, t.finishResearch))
	sdk.AddTool(server, toolOf(domain.ToolAskQuestion), guarded(t.guard, domain.ToolAskQuestion,
		func(askQuestionInput) string { return "ask the person for information" }, t.askQuestion))
}

func (t *tools) submitPlan(ctx context.Context, in submitPlanInput) (SubmitPlanResult, report, error) {
	result, err := callWork(ctx, t, in.WorkTarget, "submit plan",
		func(who identity, projectID string) (SubmitPlanResult, error) {
			return t.works.SubmitPlan(ctx, who.User, who.Client, projectID, in.WorkID,
				SubmitPlanInput{
					BaseRevision: in.BaseRevision, Summary: in.Summary,
					RequestID: in.RequestID, SmallTask: in.SmallTask,
				})
		})
	return result, report{}, err
}

func (t *tools) finishResearch(ctx context.Context, in finishResearchInput) (ResearchResult, report, error) {
	result, err := callWork(ctx, t, in.WorkTarget, "finish research",
		func(who identity, projectID string) (ResearchResult, error) {
			return t.works.FinishResearch(ctx, who.User, who.Client, projectID, in.WorkID, in.Summary)
		})
	return result, report{}, err
}

func (t *tools) askQuestion(ctx context.Context, in askQuestionInput) (QuestionResult, report, error) {
	if len(in.Questions) == 0 || len(in.Questions) > 4 {
		return QuestionResult{}, report{}, domain.ErrBadQuestion
	}
	questions, err := json.Marshal(in.Questions)
	if err != nil {
		return QuestionResult{}, report{}, domain.ErrBadQuestion
	}
	result, err := callWork(ctx, t, in.WorkTarget, "ask question",
		func(who identity, projectID string) (QuestionResult, error) {
			return t.works.AskQuestion(ctx, who.User, who.Client, projectID, in.WorkID, in.RequestID, questions)
		})
	return result, report{}, err
}
