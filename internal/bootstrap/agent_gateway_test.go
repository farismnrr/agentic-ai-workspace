package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// gateway is a fake OpenAI-compatible model that answers with a fixed script.
type gateway struct {
	// hold, when set, makes the model wait for it to close before answering.
	hold chan struct{}
	// late, by step number, builds a step from the requests so far, for a script that needs an earlier answer.
	late     map[int]func(requests []map[string]any) string
	server   *httptest.Server
	steps    []string
	requests []map[string]any
	mu       sync.Mutex
}

func newGateway(t *testing.T, steps ...string) *gateway {
	t.Helper()

	g := &gateway{steps: steps}
	g.server = httptest.NewServer(http.HandlerFunc(g.serve))
	t.Cleanup(g.server.Close)

	return g
}

func (g *gateway) serve(w http.ResponseWriter, r *http.Request) {
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

	g.mu.Lock()
	g.requests = append(g.requests, body)
	index := len(g.requests) - 1
	step := ""

	if build := g.late[index]; build != nil {
		step = build(g.requests)
	} else if index < len(g.steps) {
		step = g.steps[index]
	}
	g.mu.Unlock()

	if step == "" {
		http.Error(w, `{"error":{"message":"script ran out"}}`, http.StatusInternalServerError)

		return
	}

	if g.hold != nil {
		<-g.hold
	}

	if streaming, _ := body["stream"].(bool); streaming {
		streamStep(w, step)

		return
	}

	_, _ = w.Write([]byte(step))
}

func (g *gateway) sent(index int) []any {
	g.mu.Lock()
	defer g.mu.Unlock()

	messages, _ := g.requests[index]["messages"].([]any)

	return messages
}
