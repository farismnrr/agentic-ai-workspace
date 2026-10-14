package openai_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/openai"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func chunks(w http.ResponseWriter, lines ...string) {
	w.Header().Set("Content-Type", "text/event-stream")

	for _, line := range lines {
		_, _ = fmt.Fprintf(w, "data: %s\n\n", line)
	}
}

func TestStreamedAnswerIsAssembledAndTextIsForwardedAsItArrives(t *testing.T) {
	t.Parallel()

	var asked struct {
		StreamOptions map[string]any `json:"stream_options"`
		Stream        bool           `json:"stream"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &asked)

		chunks(w,
			`{"choices":[{"delta":{"role":"assistant","content":""}}]}`,
			`{"choices":[{"delta":{"content":"Hel"}}]}`,
			`{"choices":[{"delta":{"content":"lo"}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1",`+
				`"function":{"name":"read_file","arguments":"{\"pa"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\":\"a\"}"}}]}}]}`,
			`{"choices":[],"usage":{"prompt_tokens":20,"completion_tokens":6}}`,
			`[DONE]`,
		)
	}))
	defer server.Close()

	var pieces []string

	request := conversation()
	request.OnText = func(delta string) { pieces = append(pieces, delta) }

	reply, err := openai.New(server.URL+"/v1", "k").Complete(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}

	if !asked.Stream || asked.StreamOptions["include_usage"] != true || strings.Join(pieces, "|") != "Hel|lo" {
		t.Fatalf("a streaming request asks for usage and forwards text: %+v %v", asked, pieces)
	}

	wantAssembled(t, reply)
}

func wantAssembled(t *testing.T, reply application.Reply) {
	t.Helper()

	if len(reply.Blocks) != 2 || reply.Blocks[0].Text != "Hello" || reply.Blocks[1].ID != "c1" ||
		string(reply.Blocks[1].Input) != `{"path":"a"}` || reply.Usage.Input != 20 || reply.Usage.Output != 6 {
		t.Fatalf("answer not assembled: %+v", reply)
	}
}

func TestAGatewayThatRefusesStreamingIsAskedAgainWithoutIt(t *testing.T) {
	t.Parallel()

	var streamed []bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Stream bool `json:"stream"`
		}

		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		streamed = append(streamed, body.Stream)

		if body.Stream {
			http.Error(w, `{"error":{"message":"stream_options not supported"}}`, http.StatusBadRequest)

			return
		}

		_, _ = w.Write([]byte(answer))
	}))
	defer server.Close()

	request := conversation()
	request.OnText = func(string) { t.Error("nothing was streamed, so nothing should be forwarded") }

	reply, err := openai.New(server.URL+"/v1", "k").Complete(context.Background(), request)
	if err != nil || len(reply.Blocks) != 2 {
		t.Fatalf("the plain request should have worked: %+v %v", reply, err)
	}

	if len(streamed) != 2 || !streamed[0] || streamed[1] {
		t.Fatalf("expected a streaming try then a plain one, got %v", streamed)
	}
}
