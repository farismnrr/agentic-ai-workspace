package anthropic_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/anthropic"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func sse(w http.ResponseWriter, events ...[2]string) {
	w.Header().Set("Content-Type", "text/event-stream")

	for _, event := range events {
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event[0], event[1])
	}
}

func TestStreamedAnswerIsAssembledAndTextIsForwardedAsItArrives(t *testing.T) {
	t.Parallel()

	var asked map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &asked)

		sse(w,
			[2]string{"message_start", `{"type":"message_start","message":{"usage":{"input_tokens":25,"output_tokens":1}}}`},
			[2]string{"ping", `{"type":"ping"}`},
			[2]string{"content_block_start", `{"index":0,"content_block":{"type":"text","text":""}}`},
			[2]string{"content_block_delta", `{"index":0,"delta":{"type":"text_delta","text":"Hel"}}`},
			[2]string{"content_block_delta", `{"index":0,"delta":{"type":"text_delta","text":"lo"}}`},
			[2]string{"content_block_start", `{"index":1,"content_block":{"type":"tool_use","id":"t9","name":"read_file"}}`},
			[2]string{"content_block_delta", `{"index":1,"delta":{"type":"input_json_delta","partial_json":"{\"pa"}}`},
			[2]string{"content_block_delta", `{"index":1,"delta":{"type":"input_json_delta","partial_json":"th\":\"a\"}"}}`},
			[2]string{"message_delta", `{"delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":15}}`},
			[2]string{"message_stop", `{"type":"message_stop"}`},
		)
	}))
	defer server.Close()

	var pieces []string

	request := conversation()
	request.OnText = func(delta string) { pieces = append(pieces, delta) }

	reply, err := anthropic.New(server.URL, "k", local).Complete(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}

	if asked["stream"] != true || strings.Join(pieces, "|") != "Hel|lo" {
		t.Fatalf("text must be forwarded piece by piece on a streaming request: %v %v", asked["stream"], pieces)
	}

	if len(reply.Blocks) != 2 || reply.Blocks[0].Text != "Hello" || reply.Blocks[1].ID != "t9" ||
		string(reply.Blocks[1].Input) != `{"path":"a"}` || reply.Usage.Input != 25 || reply.Usage.Output != 15 {
		t.Fatalf("answer not assembled: %+v", reply)
	}
}

func TestAnErrorInTheMiddleOfAStreamIsReportedAndRetryable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sse(w, [2]string{"error", `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`})
	}))
	defer server.Close()

	request := conversation()
	request.OnText = func(string) {}

	_, err := anthropic.New(server.URL, "k", local).Complete(context.Background(), request)

	modelErr, ok := domain.AsModelError(err)
	if !ok || modelErr.Message != "Overloaded" || !modelErr.Retryable {
		t.Fatalf("got %+v (%v)", modelErr, err)
	}
}
