package anthropic_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/anthropic"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

// local lets the clients reach the loopback test servers.
var local = egress.Policy{Rule: egress.LocalOnly}

const answer = `{"content":[{"type":"text","text":"On it"},` +
	`{"type":"tool_use","id":"t2","name":"read_file","input":{"path":"b"}}],` +
	`"usage":{"input_tokens":12,"output_tokens":7}}`

func conversation() application.Request {
	return application.Request{
		Model: "claude-x", System: "Be kind.",
		Tools: []domain.ToolDef{{Name: "read_file", Description: "Read", Schema: json.RawMessage(`{"type":"object"}`)}},
		Messages: []domain.Message{
			{Role: domain.RoleUser, Blocks: []domain.Block{{Kind: domain.BlockText, Text: "Fix it"}}},
			{Role: domain.RoleAssistant, Blocks: []domain.Block{
				{Kind: domain.BlockText, Text: "Looking."},
				{Kind: domain.BlockToolUse, ID: "t1", Name: "read_file", Input: json.RawMessage(`{"path":"a"}`)},
			}},
			{Role: domain.RoleUser, Blocks: []domain.Block{
				{Kind: domain.BlockToolResult, ID: "t1", Text: "contents", IsError: true},
				{Kind: domain.BlockText, Text: "  "},
			}},
		},
	}
}

// serve answers like the Messages API and returns the request body it received.
func serve(t *testing.T) (*httptest.Server, *map[string]any) {
	t.Helper()

	seen := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "k" || r.Header.Get("anthropic-version") == "" {
			http.Error(w, `{"error":{"message":"bad request line"}}`, http.StatusBadRequest)

			return
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &seen)
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)

	return server, &seen
}

func TestReplyKeepsTextToolCallsAndUsage(t *testing.T) {
	t.Parallel()

	server, _ := serve(t)

	reply, err := anthropic.New(server.URL+"/", "k", local).Complete(context.Background(), conversation())
	if err != nil {
		t.Fatal(err)
	}

	if len(reply.Blocks) != 2 || reply.Blocks[1].ID != "t2" || reply.Usage.Input != 12 || reply.Usage.Output != 7 {
		t.Fatalf("reply not parsed: %+v", reply)
	}
}

func TestRequestSendsSystemPromptAndDropsBlankText(t *testing.T) {
	t.Parallel()

	server, seen := serve(t)

	if _, err := anthropic.New(server.URL, "k", local).Complete(context.Background(), conversation()); err != nil {
		t.Fatal(err)
	}

	messages, _ := (*seen)["messages"].([]any)
	last, _ := messages[len(messages)-1].(map[string]any)
	content, _ := last["content"].([]any)

	if (*seen)["system"] != "Be kind." || len(messages) != 3 || len(content) != 1 {
		t.Fatalf("blank text must be dropped and the system prompt sent: %v", *seen)
	}
}

func TestToolResultsAreEncodedWithTheirCallID(t *testing.T) {
	t.Parallel()

	server, seen := serve(t)

	if _, err := anthropic.New(server.URL, "k", local).Complete(context.Background(), conversation()); err != nil {
		t.Fatal(err)
	}

	messages, _ := (*seen)["messages"].([]any)
	last, _ := messages[len(messages)-1].(map[string]any)
	content, _ := last["content"].([]any)
	result, _ := content[0].(map[string]any)

	if result["type"] != "tool_result" || result["tool_use_id"] != "t1" || result["is_error"] != true {
		t.Fatalf("tool result not encoded: %v", result)
	}
}

func TestFailuresKeepTheProvidersMessageAndSayIfRetryingHelps(t *testing.T) {
	t.Parallel()

	statuses := map[int]bool{http.StatusTooManyRequests: true, 529: true, http.StatusUnauthorized: false}

	for status, retryable := range statuses {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"type":"x","message":"provider says no"}}`))
		}))

		_, err := anthropic.New(server.URL, "k", local).Complete(context.Background(), conversation())
		server.Close()

		modelErr, ok := domain.AsModelError(err)
		if !ok || modelErr.Message != "provider says no" || modelErr.Retryable != retryable {
			t.Errorf("HTTP %d: got %+v (%v), want retryable=%v", status, modelErr, err, retryable)
		}
	}
}

func TestRedirectsAreRefusedSoKeysStayHome(t *testing.T) {
	t.Parallel()

	elsewhere := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("the redirect target must never be called")
	}))
	defer elsewhere.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()

	if _, err := anthropic.New(server.URL, "k", local).Complete(context.Background(), conversation()); err == nil {
		t.Fatal("a redirect must fail")
	}
}

func TestEffortGoesInOutputConfigOnlyWhenChosen(t *testing.T) {
	t.Parallel()

	server, seen := serve(t)
	request := conversation()
	request.Effort = domain.EffortXHigh

	if _, err := anthropic.New(server.URL, "k", local).Complete(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	config, _ := (*seen)["output_config"].(map[string]any)
	if config["effort"] != "xhigh" {
		t.Fatalf("output_config = %v, want effort xhigh", (*seen)["output_config"])
	}

	request.Effort = domain.EffortDefault
	*seen = map[string]any{}

	if _, err := anthropic.New(server.URL, "k", local).Complete(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	if _, present := (*seen)["output_config"]; present {
		t.Fatal("no effort chosen must leave output_config out")
	}
}
