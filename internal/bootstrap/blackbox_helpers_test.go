package bootstrap

import (
	"encoding/json"
	"testing"
	"time"
)

func (a agentRig) startTeam(t *testing.T, prompt string) string {
	t.Helper()

	created := do(t, a.env.app, "POST", "/api/tasks",
		`{"projectId":"`+a.project+`","prompt":"`+prompt+`","team":true}`, jsonCookie(a.cookie))
	if created.Status != 201 {
		t.Fatalf("task not created: %d %s", created.Status, created.Body)
	}

	id, _ := created.JSON(t)["id"].(string)

	return id
}

// finishLive follows the task, saying yes to every approval, until it is done. Real services are slow.
func finishLive(t *testing.T, a agentRig, id string) {
	t.Helper()

	deadline := time.Now().Add(12 * time.Minute)

	for time.Now().Before(deadline) {
		switch a.status(t, id) {
		case "done", "review":
			return
		case "needs-input":
			a.allowPending(t, id)
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("task never finished, it is %q", a.status(t, id))
}

// toolResults is every tool answer the model was shown, in order, as one text per model request.
func (a agentRig) toolResults(t *testing.T) []string {
	t.Helper()

	a.gateway.mu.Lock()
	defer a.gateway.mu.Unlock()

	texts := make([]string, 0, len(a.gateway.requests))

	for _, request := range a.gateway.requests {
		raw, _ := json.Marshal(request["messages"])
		texts = append(texts, string(raw))
	}

	return texts
}
