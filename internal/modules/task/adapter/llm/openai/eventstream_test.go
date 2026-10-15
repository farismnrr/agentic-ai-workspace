package openai_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/openai"
)

// 9router answers every request to a Claude model with server-sent events, even one that did not ask for a stream.
const eventStream = `data: {"choices":[{"index":0,"delta":{"role":"assistant"}}]}

data: {"choices":[{"index":0,"delta":{"content":"The summary"}}]}

data: {"choices":[{"index":0,"delta":{"content":" so far."}}]}

data: {"choices":[],"usage":{"prompt_tokens":40,"completion_tokens":6}}

data: [DONE]

`

func TestAnEventStreamAnswerToAPlainRequestIsStillRead(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(eventStream))
	}))
	t.Cleanup(server.Close)

	reply, err := openai.New(server.URL+"/v1", "k", local).Complete(context.Background(), conversation())
	if err != nil {
		t.Fatal(err)
	}

	if len(reply.Blocks) != 1 || reply.Blocks[0].Text != "The summary so far." {
		t.Fatalf("the stream must be assembled into one answer: %+v", reply)
	}

	if reply.Usage.Input != 40 || reply.Usage.Output != 6 {
		t.Fatalf("the stream must be assembled into one answer: %+v", reply)
	}
}
