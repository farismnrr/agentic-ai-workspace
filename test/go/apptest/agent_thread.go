package apptest

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ThreadEvent is one event returned by a task thread API response.
type ThreadEvent struct {
	ID       string   `json:"id"`
	URL      string   `json:"url"`
	Kind     string   `json:"kind"`
	Text     string   `json:"text"`
	Action   string   `json:"action"`
	Resolved string   `json:"resolved"`
	Agent    string   `json:"agent"`
	Files    []string `json:"files"`
	Seq      int64    `json:"seq"`
	Version  int64    `json:"version"`
	CanUndo  bool     `json:"canUndo"`
}

// WaitApproving follows a task until it reaches a status, accepting each prompt.
func (a AgentRig) WaitApproving(t *testing.T, id, want string) {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		status := a.Status(t, id)
		switch status {
		case want:
			return
		case "needs-input":
			a.AllowPending(t, id)
		case "failed", "review", "done":
			t.Fatalf("task reached %q instead of %q; %s", status, want, a.failureEvents(t, id))
		}
		time.Sleep(requestPollInterval)
	}
	t.Fatalf("task never reached %q, it is %q; %s", want, a.Status(t, id), a.failureEvents(t, id))
}

// WaitApprovals polls until the thread holds the requested number of approvals.
func (a AgentRig) WaitApprovals(t *testing.T, id string, count int) {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		approvals := 0
		for _, event := range a.Thread(t, id) {
			if event.Kind == "approval" {
				approvals++
			}
		}
		if approvals >= count {
			return
		}
		time.Sleep(requestPollInterval)
	}
	t.Fatalf("the agent never asked %d times", count)
}

// AllowPending approves the newest unresolved approval event.
func (a AgentRig) AllowPending(t *testing.T, id string) {
	t.Helper()
	events := a.Thread(t, id)
	for index := len(events) - 1; index >= 0; index-- {
		if events[index].Kind == "approval" && events[index].Resolved == "" {
			a.Post(t, "/api/tasks/"+id+"/events/"+events[index].ID+"/answer", `{"allow":true}`)
			return
		}
	}
}

// Thread returns the task's current events.
func (a AgentRig) Thread(t *testing.T, id string) []ThreadEvent {
	t.Helper()
	var events []ThreadEvent
	body := Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
		JSONCookie(a.Cookie)).Body
	if err := json.Unmarshal(body, &events); err != nil {
		t.Fatal(err)
	}
	return events
}

// failureEvents keeps CI annotations short enough to show the actual failure.
func (a AgentRig) failureEvents(t *testing.T, id string) string {
	t.Helper()
	var details []string
	for _, event := range a.Thread(t, id) {
		if event.Kind != "error" && event.Kind != "approval" && event.Kind != "action" {
			continue
		}
		value := fmt.Sprintf("%s/%s: %s", event.Kind, event.Action, event.Text)
		if len(value) > 200 {
			value = value[:200]
		}
		details = append(details, value)
	}
	if len(details) > 8 {
		details = details[len(details)-8:]
	}
	return strings.Join(details, " | ")
}
