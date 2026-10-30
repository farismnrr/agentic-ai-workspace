package apptest

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

const (
	maxDiagnosticEvents = 24
	maxDiagnosticText   = 240
	maxDiagnosticCmd    = 160
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
		switch a.Status(t, id) {
		case want:
			return
		case "needs-input":
			a.AllowPending(t, id)
		}
		time.Sleep(requestPollInterval)
	}
	t.Fatalf("task never reached %q: %s", want, a.Diagnostics(t, id))
}

// Diagnostics returns a bounded task status and event summary for test failures.
func (a AgentRig) Diagnostics(t *testing.T, id string) string {
	t.Helper()
	var events []map[string]any
	body := Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
		JSONCookie(a.Cookie)).Body
	if err := json.Unmarshal(body, &events); err != nil {
		t.Fatalf("decode task diagnostics: %v", err)
	}
	if len(events) > maxDiagnosticEvents {
		events = events[len(events)-maxDiagnosticEvents:]
	}
	lines := make([]string, 0, len(events))
	for _, event := range events {
		text, _ := event["text"].(string)
		if len(text) > maxDiagnosticText {
			text = text[:maxDiagnosticText] + "…"
		}
		kind, _ := event["kind"].(string)
		line := kind + ": " + text
		run, _ := event["run"].(map[string]any)
		command, _ := run["command"].(string)
		if command != "" {
			if len(command) > maxDiagnosticCmd {
				command = command[:maxDiagnosticCmd] + "…"
			}
			exitCode, _ := run["exitCode"].(float64)
			timedOut, _ := run["timedOut"].(bool)
			line += fmt.Sprintf(" [command=%q exit=%.0f timeout=%t]", command, exitCode, timedOut)
		}
		lines = append(lines, line)
	}
	return fmt.Sprintf("status=%s events=%s", a.Status(t, id), strings.Join(lines, " | "))
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
