package mcp

import (
	"context"
	"encoding/json"
)

func (f *fakeWorkSessions) FinishResearch(
	_ context.Context, _, _, _, workID, summary string,
) (ResearchResult, error) {
	return ResearchResult{Summary: summary, Missing: []string{"save sourced research"}}, nil
}

func (f *fakeWorkSessions) AskQuestion(
	_ context.Context, _, _, _, workID, requestID string, _ json.RawMessage,
) (QuestionResult, error) {
	f.state.Questions.Open = []WorkQuestion{{
		ID: "question-1", EventID: "event-question-1", State: "open",
		Form: []QuestionForm{{Text: "Which name?", Type: "text"}},
	}}
	f.state.NextAction = "wait_for_answers"
	return QuestionResult{QuestionID: "question-1", EventID: "event-question-1", State: "open"}, nil
}
