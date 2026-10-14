package bootstrap

import (
	"encoding/json"
	"testing"
	"time"
)

// waitApproving follows a run to a status, allowing every step that asks first.
func (a agentRig) waitApproving(t *testing.T, id, want string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		switch a.status(t, id) {
		case want:
			return
		case "needs-input":
			a.allowPending(t, id)
		}

		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("task never reached %q, it is %q", want, a.status(t, id))
}

// waitApprovals polls until the thread holds n approval requests.
func (a agentRig) waitApprovals(t *testing.T, id string, n int) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		count := 0

		for _, event := range a.thread(t, id) {
			if event.Kind == "approval" {
				count++
			}
		}

		if count >= n {
			return
		}

		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("the agent never asked %d times", n)
}

// allowPending says yes to the newest approval that is still waiting.
func (a agentRig) allowPending(t *testing.T, id string) {
	t.Helper()

	events := a.thread(t, id)

	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Kind == "approval" && events[i].Resolved == "" {
			a.post(t, "/api/tasks/"+id+"/events/"+events[i].ID+"/answer", `{"allow":true}`)

			return
		}
	}
}

type threadEvent struct {
	ID       string   `json:"id"`
	URL      string   `json:"url"`
	Kind     string   `json:"kind"`
	Text     string   `json:"text"`
	Resolved string   `json:"resolved"`
	Files    []string `json:"files"`
	Seq      int64    `json:"seq"`
	CanUndo  bool     `json:"canUndo"`
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
