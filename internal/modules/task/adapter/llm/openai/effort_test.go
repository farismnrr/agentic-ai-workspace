package openai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/openai"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// bodyOf sends one request at the given effort and returns the JSON the server received.
func bodyOf(t *testing.T, effort domain.Effort) map[string]any {
	t.Helper()

	seen := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &seen)
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)

	request := conversation()
	request.Effort = effort

	if _, err := openai.New(server.URL+"/v1", "k", local).Complete(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	return seen
}

func TestEffortIsSentAsReasoningEffortOnlyWhenChosen(t *testing.T) {
	t.Parallel()

	if got := bodyOf(t, domain.EffortHigh)["reasoning_effort"]; got != "high" {
		t.Fatalf("reasoning_effort = %v, want high", got)
	}

	if _, present := bodyOf(t, domain.EffortDefault)["reasoning_effort"]; present {
		t.Fatal("no effort chosen must leave the field out so the provider keeps its own default")
	}
}
