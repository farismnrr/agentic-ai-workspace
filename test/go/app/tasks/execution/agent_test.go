package execution_test

import (
	"os"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestTaskRunsFromRequestToReviewAndCanBeAcceptedAndUndone(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t,
		apptest.Plan(),
		apptest.ToolCall("w1", "write_file", map[string]string{"path": "hello.txt", "content": "hi"}),
		apptest.FinishCall("Made hello.txt"),
	)
	id := a.Start(t, "Make a hello file")
	a.WaitApproving(t, id, "review")

	if data := a.Read("hello.txt"); data != "hi" {
		t.Fatalf("the agent should have written the file, got %q", data)
	}

	var changed apptest.ThreadEvent

	for _, event := range a.Thread(t, id) {
		if len(event.Files) > 0 {
			changed = event
		}
	}

	if !changed.CanUndo || changed.Kind != "action" {
		t.Fatalf("a file change must be an undoable action: %+v", changed)
	}

	task := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id, "", apptest.JSONCookie(a.Cookie)).Body
	apptest.MustContain(t, task, `"summary":"Made hello.txt"`, `"proven":false`, "Not verified", `"title":"Do it"`)

	if a.Post(t, "/api/tasks/"+id+"/events/"+changed.ID+"/undo", "") != 204 {
		t.Fatal("undo should succeed")
	}

	if _, err := os.Stat(a.File("hello.txt")); !os.IsNotExist(err) {
		t.Fatal("undo must remove the file the agent created")
	}

	if a.Post(t, "/api/tasks/"+id+"/accept", "") != 204 || a.Status(t, id) != "done" {
		t.Fatal("reviewed work can be accepted")
	}

	usage := apptest.Do(t, a.Environment.App, "GET", "/api/usage", "", apptest.JSONCookie(a.Cookie)).Body
	apptest.MustContain(t, usage, `"model":"fake-model"`, `"tasks":1`, `"inputTokens":30`)
}

func TestEveryChangeStopsForApprovalInTheRealApp(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t,
		apptest.ToolCall("w1", "write_file", map[string]string{"path": "old.txt", "content": "x"}),
		apptest.ToolCall("d1", "delete_file", map[string]string{"path": "old.txt"}),
		apptest.FinishCall("Cleaned up"),
	)
	id := a.Start(t, "Create and remove a file")
	a.WaitFor(t, id, "needs-input")

	if _, err := os.Stat(a.File("old.txt")); !os.IsNotExist(err) {
		t.Fatalf("writing must wait for a yes, but the file exists: %v", err)
	}

	a.AllowPending(t, id)
	a.WaitApprovals(t, id, 2)

	var approval apptest.ThreadEvent

	for _, event := range a.Thread(t, id) {
		if event.Kind == "approval" && event.Resolved == "" {
			approval = event
		}
	}

	if _, err := os.Stat(a.File("old.txt")); err != nil || !strings.Contains(approval.Text, "old.txt") {
		t.Fatalf("the file must still exist while the delete waits: %v %+v", err, approval)
	}

	if a.Post(t, "/api/tasks/"+id+"/events/"+approval.ID+"/answer", `{"allow":true}`) != 204 {
		t.Fatal("answering should succeed")
	}

	a.WaitFor(t, id, "review")

	if _, err := os.Stat(a.File("old.txt")); !os.IsNotExist(err) {
		t.Fatal("after approval the delete must happen")
	}
}

func TestFollowUpContinuesTheConversationWithValidMessages(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t, apptest.FinishCall("First"), apptest.FinishCall("Second"))
	id := a.Start(t, "Do something")
	a.WaitFor(t, id, "done")

	if a.Post(t, "/api/tasks/"+id+"/messages", `{"text":"Also the footer"}`) != 204 {
		t.Fatal("a follow-up on a finished conversation should be accepted")
	}

	a.WaitFor(t, id, "done")

	sent := a.Gateway.Sent(1)
	last, _ := sent[len(sent)-1].(map[string]any)
	before, _ := sent[len(sent)-2].(map[string]any)

	if before["role"] != "tool" || last["role"] != "user" || last["content"] != "Also the footer" {
		t.Fatalf("the finish result must come before the new message: %v / %v", before, last)
	}
}

func TestThreadEntriesCarryIncreasingSequenceNumbersForResuming(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t, apptest.FinishCall("Done"))
	id := a.Start(t, "Count me")
	a.WaitFor(t, id, "done")

	events := a.Thread(t, id)
	if len(events) < 2 {
		t.Fatalf("expected a request and an answer, got %d", len(events))
	}

	for i, event := range events {
		if event.Seq != int64(i+1) {
			t.Fatalf("entry %d has sequence %d, want %d", i, event.Seq, i+1)
		}
	}
}
