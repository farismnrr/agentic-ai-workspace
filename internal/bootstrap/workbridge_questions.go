package bootstrap

import (
	"context"
	"encoding/json"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (a appWorks) FinishResearch(ctx context.Context, user, client, project, workID, summary string) (
	codingmcp.ResearchResult, error,
) {
	if len([]rune(summary)) > taskdomain.MaxWorkSummary {
		return codingmcp.ResearchResult{}, taskdomain.ErrTooLarge
	}
	summary, _ = taskdomain.Redact(summary)
	ready, missing, err := a.works.FinishResearch(ctx, user, client, project, workID)
	if err != nil {
		return codingmcp.ResearchResult{}, workBridgeError("finish research", err)
	}
	return codingmcp.ResearchResult{Ready: ready, Missing: missing, Summary: summary}, nil
}

func (a appWorks) AskQuestion(ctx context.Context, user, client, project, workID, requestID string,
	questions json.RawMessage,
) (codingmcp.QuestionResult, error) {
	form, err := taskapp.ParseWorkQuestionForm(questions)
	if err != nil {
		return codingmcp.QuestionResult{}, workBridgeError("ask question", err)
	}
	result, err := a.works.AskQuestion(ctx, user, client, project, workID, requestID, form)
	if err != nil {
		return codingmcp.QuestionResult{}, workBridgeError("ask question", err)
	}
	return codingmcp.QuestionResult{QuestionID: result.ID, EventID: result.EventID, State: result.State}, nil
}

func stateQuestions(in taskapp.QuestionGroups) codingmcp.QuestionGroups {
	return codingmcp.QuestionGroups{
		Open: stateQuestionList(in.Open), Answered: stateQuestionList(in.Answered),
	}
}

func stateQuestionList(in []taskapp.WorkQuestion) []codingmcp.WorkQuestion {
	out := make([]codingmcp.WorkQuestion, 0, len(in))
	for _, question := range in {
		form := make([]codingmcp.QuestionForm, 0, len(question.Form))
		for _, item := range question.Form {
			mapped := codingmcp.QuestionForm{Header: item.Header, Text: item.Text, Type: string(item.Type)}
			for _, option := range item.Options {
				mapped.Options = append(mapped.Options, codingmcp.QuestionOption{
					Label: option.Label, Description: option.Description, Preview: option.Preview,
				})
			}
			form = append(form, mapped)
		}
		out = append(out, codingmcp.WorkQuestion{
			ID: question.ID, EventID: question.EventID, Form: form, Answers: question.Answers, AskedAt: question.AskedAt,
			State: question.State, Summary: question.Summary,
		})
	}
	return out
}
