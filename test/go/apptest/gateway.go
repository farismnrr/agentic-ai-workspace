package apptest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Gateway is a scripted OpenAI-compatible model used by app integration tests.
type Gateway struct {
	Hold     chan struct{}
	Late     map[int]func(requests []map[string]any) string
	Server   *httptest.Server
	Steps    []string
	Requests []map[string]any
	Mu       sync.Mutex
}

// NewGateway starts a local model server that returns the supplied replies.
func NewGateway(t *testing.T, steps ...string) *Gateway {
	t.Helper()
	gateway := &Gateway{Steps: steps, Late: map[int]func([]map[string]any) string{}}
	gateway.Server = httptest.NewServer(http.HandlerFunc(gateway.serve))
	t.Cleanup(gateway.Server.Close)
	return gateway
}

func (g *Gateway) serve(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer k" {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
		return
	}
	if r.URL.Path == "/models" {
		_, _ = w.Write([]byte(`{"data":[{"id":"fake-model"}]}`))
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	g.Mu.Lock()
	g.Requests = append(g.Requests, body)
	index := len(g.Requests) - 1
	step := ""
	if build := g.Late[index]; build != nil {
		step = build(g.Requests)
	} else if index < len(g.Steps) {
		step = g.Steps[index]
	}
	g.Mu.Unlock()
	if step == "" {
		http.Error(w, `{"error":{"message":"script ran out"}}`, http.StatusInternalServerError)
		return
	}
	if g.Hold != nil {
		<-g.Hold
	}
	if streaming, _ := body["stream"].(bool); streaming {
		streamStep(w, step)
		return
	}
	_, _ = w.Write([]byte(step))
}

// Sent returns the messages sent with one model request.
func (g *Gateway) Sent(index int) []any {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	messages, _ := g.Requests[index]["messages"].([]any)
	return messages
}

// ToolCall encodes one OpenAI-compatible tool-call reply.
func ToolCall(id, name string, args any) string {
	encoded, _ := json.Marshal(args)
	function := map[string]string{"name": name, "arguments": string(encoded)}
	call := []map[string]any{{"id": id, "type": "function", "function": function}}
	body, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]any{"content": nil, "tool_calls": call}}},
		"usage": map[string]int{
			"prompt_tokens": promptTokenCount, "completion_tokens": completionTokenCount,
		},
	})
	return string(body)
}

// Plan returns a model reply that makes a one-step plan.
func Plan() string {
	return ToolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
		{"title": "Do it", "state": "active"},
	}})
}

// FinishCall returns a model reply that ends a task with a summary.
func FinishCall(summary string) string {
	return ToolCall("f1", "finish", map[string]any{"summary": summary, "checks": []string{"Looked it over"}})
}
