package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func (a agentRig) read(name string) string {
	data, err := os.ReadFile(a.file(name))
	if err != nil {
		return ""
	}

	return string(data)
}

func (a agentRig) file(id string) string {
	return filepath.Join(a.env.dataDir, "projects", a.project, id)
}

func TestTaskRunsFromRequestToReviewAndCanBeAcceptedAndUndone(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t,
		plan(),
		toolCall("w1", "write_file", map[string]string{"path": "hello.txt", "content": "hi"}),
		finishCall("Made hello.txt"),
	)
	id := a.start(t, "Make a hello file")
	a.waitApproving(t, id, "review")

	if data := a.read("hello.txt"); data != "hi" {
		t.Fatalf("the agent should have written the file, got %q", data)
	}

	var changed threadEvent

	for _, event := range a.thread(t, id) {
		if len(event.Files) > 0 {
			changed = event
		}
	}

	if !changed.CanUndo || changed.Kind != "action" {
		t.Fatalf("a file change must be an undoable action: %+v", changed)
	}

	task := do(t, a.env.app, "GET", "/api/tasks/"+id, "", jsonCookie(a.cookie)).Body
	mustContain(t, task, `"summary":"Made hello.txt"`, `"proven":false`, "Not verified", `"title":"Do it"`)

	if a.post(t, "/api/tasks/"+id+"/events/"+changed.ID+"/undo", "") != 204 {
		t.Fatal("undo should succeed")
	}

	if _, err := os.Stat(a.file("hello.txt")); !os.IsNotExist(err) {
		t.Fatal("undo must remove the file the agent created")
	}

	if a.post(t, "/api/tasks/"+id+"/accept", "") != 204 || a.status(t, id) != "done" {
		t.Fatal("reviewed work can be accepted")
	}

	usage := do(t, a.env.app, "GET", "/api/usage", "", jsonCookie(a.cookie)).Body
	mustContain(t, usage, `"model":"fake-model"`, `"tasks":1`, `"inputTokens":30`)
}

func TestEveryChangeStopsForApprovalInTheRealApp(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t,
		toolCall("w1", "write_file", map[string]string{"path": "old.txt", "content": "x"}),
		toolCall("d1", "delete_file", map[string]string{"path": "old.txt"}),
		finishCall("Cleaned up"),
	)
	id := a.start(t, "Create and remove a file")
	a.waitFor(t, id, "needs-input")

	if _, err := os.Stat(a.file("old.txt")); !os.IsNotExist(err) {
		t.Fatalf("writing must wait for a yes, but the file exists: %v", err)
	}

	a.allowPending(t, id)
	a.waitApprovals(t, id, 2)

	var approval threadEvent

	for _, event := range a.thread(t, id) {
		if event.Kind == "approval" && event.Resolved == "" {
			approval = event
		}
	}

	if _, err := os.Stat(a.file("old.txt")); err != nil || !strings.Contains(approval.Text, "old.txt") {
		t.Fatalf("the file must still exist while the delete waits: %v %+v", err, approval)
	}

	if a.post(t, "/api/tasks/"+id+"/events/"+approval.ID+"/answer", `{"allow":true}`) != 204 {
		t.Fatal("answering should succeed")
	}

	a.waitFor(t, id, "review")

	if _, err := os.Stat(a.file("old.txt")); !os.IsNotExist(err) {
		t.Fatal("after approval the delete must happen")
	}
}

func TestFollowUpContinuesTheConversationWithValidMessages(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t, finishCall("First"), finishCall("Second"))
	id := a.start(t, "Do something")
	a.waitFor(t, id, "done")

	if a.post(t, "/api/tasks/"+id+"/messages", `{"text":"Also the footer"}`) != 204 {
		t.Fatal("a follow-up on a finished conversation should be accepted")
	}

	a.waitFor(t, id, "done")

	sent := a.gateway.sent(1)
	last, _ := sent[len(sent)-1].(map[string]any)
	before, _ := sent[len(sent)-2].(map[string]any)

	if before["role"] != "tool" || last["role"] != "user" || last["content"] != "Also the footer" {
		t.Fatalf("the finish result must come before the new message: %v / %v", before, last)
	}
}

func TestThreadEntriesCarryIncreasingSequenceNumbersForResuming(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t, finishCall("Done"))
	id := a.start(t, "Count me")
	a.waitFor(t, id, "done")

	events := a.thread(t, id)
	if len(events) < 2 {
		t.Fatalf("expected a request and an answer, got %d", len(events))
	}

	for i, event := range events {
		if event.Seq != int64(i+1) {
			t.Fatalf("entry %d has sequence %d, want %d", i, event.Seq, i+1)
		}
	}
}
