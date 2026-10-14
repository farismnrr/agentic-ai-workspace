package openai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/openai"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const answer = `{"choices":[{"message":{"content":null,"tool_calls":[` +
	`{"id":"c9","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"b\"}"}},` +
	`{"id":"c10","type":"function","function":{"name":"read_file","arguments":"not json"}}]}}],` +
	`"usage":{"prompt_tokens":20,"completion_tokens":3}}`

func conversation() application.Request {
	return application.Request{
		Model: "gpt-x", System: "Be kind.",
		Tools: []domain.ToolDef{{Name: "read_file", Description: "Read"}},
		Messages: []domain.Message{
			{Role: domain.RoleUser, Blocks: []domain.Block{{Kind: domain.BlockText, Text: "Fix it"}}},
			{Role: domain.RoleAssistant, Blocks: []domain.Block{
				{Kind: domain.BlockToolUse, ID: "t1", Name: "read_file", Input: json.RawMessage(`{"path":"a"}`)},
			}},
			{Role: domain.RoleUser, Blocks: []domain.Block{
				{Kind: domain.BlockToolResult, ID: "t1", Text: "contents"},
				{Kind: domain.BlockText, Text: "Also the footer"},
			}},
		},
	}
}

type sent struct {
	Messages []map[string]any `json:"messages"`
}

// serve answers like Chat Completions and returns the request body it received.
func serve(t *testing.T) (*httptest.Server, *sent) {
	t.Helper()

	seen := &sent{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer k" {
			http.Error(w, `{"error":{"message":"bad request line"}}`, http.StatusBadRequest)

			return
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, seen)
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)

	return server, seen
}

func TestReplyKeepsToolCallsUsageAndRepairsBadArguments(t *testing.T) {
	t.Parallel()

	server, _ := serve(t)

	reply, err := openai.New(server.URL+"/v1", "k").Complete(context.Background(), conversation())
	if err != nil {
		t.Fatal(err)
	}

	if len(reply.Blocks) != 2 || reply.Usage.Input != 20 || string(reply.Blocks[0].Input) != `{"path":"b"}` {
		t.Fatalf("reply not parsed: %+v", reply)
	}

	if !json.Valid(reply.Blocks[1].Input) {
		t.Fatalf("unreadable arguments must still become valid JSON: %s", reply.Blocks[1].Input)
	}
}

func TestToolResultsBecomeToolMessagesBeforeTheUsersText(t *testing.T) {
	t.Parallel()

	server, seen := serve(t)

	if _, err := openai.New(server.URL+"/v1", "k").Complete(context.Background(), conversation()); err != nil {
		t.Fatal(err)
	}

	roles := make([]string, 0, len(seen.Messages))
	for _, message := range seen.Messages {
		role, _ := message["role"].(string)
		roles = append(roles, role)
	}

	want := []string{"system", "user", "assistant", "tool", "user"}
	if !slices.Equal(roles, want) || seen.Messages[3]["tool_call_id"] != "t1" {
		t.Fatalf("roles = %v, want %v; tool message %v", roles, want, seen.Messages[3])
	}
}

func TestFailuresKeepTheProvidersMessageAndSayIfRetryingHelps(t *testing.T) {
	t.Parallel()

	statuses := map[int]bool{http.StatusTooManyRequests: true, http.StatusBadGateway: true, http.StatusForbidden: false}

	for status, retryable := range statuses {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"message":"gateway says no"}}`))
		}))

		_, err := openai.New(server.URL, "k").Complete(context.Background(), conversation())
		server.Close()

		modelErr, ok := domain.AsModelError(err)
		if !ok || modelErr.Message != "gateway says no" || modelErr.Retryable != retryable {
			t.Errorf("HTTP %d: got %+v (%v), want retryable=%v", status, modelErr, err, retryable)
		}
	}
}
