package mcp

import (
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func checkResearchAndQuestionTools(t *testing.T, session *sdk.ClientSession, workID string) {
	t.Helper()
	research, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "finish_research", Arguments: map[string]any{
		"work_id": workID,
	}})
	requireToolSuccess(t, research, err)
	assertResearchBlocked(t, research)
	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "ask_question", Arguments: map[string]any{
		"work_id": workID, "request_id": "question-request-1",
		"questions": []any{map[string]any{"question": "Which name?", "type": "text"}},
	}})
	requireToolSuccess(t, result, err)
	assertQuestionCreated(t, result)
	state, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: "get_work_state", Arguments: map[string]any{"work_id": workID},
	})
	requireToolSuccess(t, state, err)
	assertOpenQuestionState(t, state)
}

func assertResearchBlocked(t *testing.T, result *sdk.CallToolResult) {
	t.Helper()
	body, ok := result.StructuredContent.(map[string]any)
	if !ok || body["ready"] != false || body["missing"] == nil {
		t.Fatalf("thin research result = %#v", result.StructuredContent)
	}
}

func assertQuestionCreated(t *testing.T, result *sdk.CallToolResult) {
	t.Helper()
	body, ok := result.StructuredContent.(map[string]any)
	if !ok || body["question_id"] != "question-1" || body["event_id"] != "event-question-1" {
		t.Fatalf("ask_question result = %#v", result.StructuredContent)
	}
}

func assertOpenQuestionState(t *testing.T, result *sdk.CallToolResult) {
	t.Helper()
	body, ok := result.StructuredContent.(map[string]any)
	if !ok || body["next_action"] != "wait_for_answers" {
		t.Fatalf("question resume state = %#v", result.StructuredContent)
	}
	groups, ok := body["questions"].(map[string]any)
	if !ok {
		t.Fatalf("question groups omitted = %#v", result.StructuredContent)
	}
	open, ok := groups["open"].([]any)
	if !ok || len(open) != 1 {
		t.Fatalf("open questions omitted from resumed state = %#v", result.StructuredContent)
	}
}
