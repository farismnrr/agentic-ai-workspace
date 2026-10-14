package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// gateway is a fake OpenAI-compatible model that answers with a fixed script.
type gateway struct {
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
	g.mu.Unlock()

	if index >= len(g.steps) {
		http.Error(w, `{"error":{"message":"script ran out"}}`, http.StatusInternalServerError)

		return
	}

	_, _ = w.Write([]byte(g.steps[index]))
}

func (g *gateway) sent(index int) []any {
	g.mu.Lock()
	defer g.mu.Unlock()

	messages, _ := g.requests[index]["messages"].([]any)

	return messages
}

func toolCall(id, name string, args any) string {
	encoded, _ := json.Marshal(args)
	function := map[string]string{"name": name, "arguments": string(encoded)}
	call := []map[string]any{{"id": id, "type": "function", "function": function}}
	body, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]any{"content": nil, "tool_calls": call}}},
		"usage":   map[string]int{"prompt_tokens": 10, "completion_tokens": 5},
	})

	return string(body)
}

func planStep(title, state string) map[string]string {
	return map[string]string{"title": title, "state": state}
}

func plan() string {
	return toolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{planStep("Do it", "active")}})
}

func finishCall(summary string) string {
	return toolCall("f1", "finish", map[string]any{"summary": summary, "checks": []string{"Looked it over"}})
}

// agentRig is a signed-in person with a verified provider, a default model and a project.
type agentRig struct {
	env     testEnv
	gateway *gateway
	cookie  string
	project string
}

func newAgentRig(t *testing.T, steps ...string) agentRig {
	t.Helper()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)
	gw := newGateway(t, steps...)

	providerID, _ := addProvider(t, env, cookie, gw.server.URL, "k").JSON(t)["id"].(string)
	do(t, env.app, "POST", "/api/providers/"+providerID+"/verify", "", jsonCookie(cookie))
	do(t, env.app, "PUT", "/api/preferences",
		`{"defaultProviderId":"`+providerID+`","defaultModel":"fake-model","rules":[]}`, jsonCookie(cookie))

	project, _ := do(t, env.app, "POST", "/api/projects", `{"name":"Bakery"}`, jsonCookie(cookie)).JSON(t)["id"].(string)

	return agentRig{env: env, gateway: gw, cookie: cookie, project: project}
}

func (a agentRig) start(t *testing.T, prompt string) string {
	t.Helper()

	body := `{"projectId":"` + a.project + `","prompt":"` + prompt + `"}`

	created := do(t, a.env.app, "POST", "/api/tasks", body, jsonCookie(a.cookie))
	if created.Status != 201 {
		t.Fatalf("task not created: %d %s", created.Status, created.Body)
	}

	id, _ := created.JSON(t)["id"].(string)

	return id
}

func (a agentRig) status(t *testing.T, id string) string {
	t.Helper()

	status, _ := do(t, a.env.app, "GET", "/api/tasks/"+id, "", jsonCookie(a.cookie)).JSON(t)["status"].(string)

	return status
}

// waitFor polls until the task reaches a status, which is how the app follows a run.
func (a agentRig) waitFor(t *testing.T, id, want string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		if a.status(t, id) == want {
			return
		}

		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("task never reached %q, it is %q", want, a.status(t, id))
}

type threadEvent struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Text    string   `json:"text"`
	Files   []string `json:"files"`
	CanUndo bool     `json:"canUndo"`
}

func (a agentRig) thread(t *testing.T, id string) []threadEvent {
	t.Helper()

	var events []threadEvent

	body := do(t, a.env.app, "GET", "/api/tasks/"+id+"/events", "", jsonCookie(a.cookie)).Body
	if err := json.Unmarshal(body, &events); err != nil {
		t.Fatal(err)
	}

	return events
}

func (a agentRig) post(t *testing.T, path, body string) int {
	t.Helper()

	return do(t, a.env.app, "POST", path, body, jsonCookie(a.cookie)).Status
}
