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
	a.waitFor(t, id, "review")

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

func TestRiskyStepsStopForApprovalInTheRealApp(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t,
		toolCall("w1", "write_file", map[string]string{"path": "old.txt", "content": "x"}),
		toolCall("d1", "delete_file", map[string]string{"path": "old.txt"}),
		finishCall("Cleaned up"),
	)
	id := a.start(t, "Create and remove a file")
	a.waitFor(t, id, "needs-input")

	var approval threadEvent

	for _, event := range a.thread(t, id) {
		if event.Kind == "approval" {
			approval = event
		}
	}

	if _, err := os.Stat(a.file("old.txt")); err != nil || !strings.Contains(approval.Text, "old.txt") {
		t.Fatalf("the file must still exist while the agent waits: %v %+v", err, approval)
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
	a.waitFor(t, id, "review")

	if a.post(t, "/api/tasks/"+id+"/messages", `{"text":"Also the footer"}`) != 204 {
		t.Fatal("a follow-up on reviewed work should be accepted")
	}

	a.waitFor(t, id, "review")

	sent := a.gateway.sent(1)
	last, _ := sent[len(sent)-1].(map[string]any)
	before, _ := sent[len(sent)-2].(map[string]any)

	if before["role"] != "tool" || last["role"] != "user" || last["content"] != "Also the footer" {
		t.Fatalf("the finish result must come before the new message: %v / %v", before, last)
	}
}

func TestTaskNeedsAModelAndStaysPrivate(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	cookie, _ := signedInClient(t, env.app)
	project, _ := do(t, env.app, "POST", "/api/projects", `{"name":"Bakery"}`, jsonCookie(cookie)).JSON(t)["id"].(string)

	noModel := do(t, env.app, "POST", "/api/tasks", `{"projectId":"`+project+`","prompt":"Hi"}`, jsonCookie(cookie))
	if noModel.Status != 409 || !strings.Contains(string(noModel.Body), "no_model") {
		t.Fatalf("without a provider the person is told to add one: %d %s", noModel.Status, noModel.Body)
	}

	unknown := do(t, env.app, "POST", "/api/tasks", `{"projectId":"nope","prompt":"Hi"}`, jsonCookie(cookie))
	if unknown.Status != 400 {
		t.Fatalf("an unknown project is refused, got %d", unknown.Status)
	}
}

func TestAnotherPersonCannotSeeOrControlATask(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t, finishCall("Done"))
	id := a.start(t, "Private work")
	a.waitFor(t, id, "review")

	other := cookieOf(t, do(t, a.env.app, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`, jsonHeaders()))

	for _, path := range []string{"/api/tasks/" + id, "/api/tasks/" + id + "/events"} {
		if got := do(t, a.env.app, "GET", path, "", jsonCookie(other)).Status; got != 404 {
			t.Errorf("GET %s by someone else = %d, want 404", path, got)
		}
	}

	if got := do(t, a.env.app, "POST", "/api/tasks/"+id+"/accept", "", jsonCookie(other)).Status; got != 404 {
		t.Errorf("accepting someone else's task = %d, want 404", got)
	}

	if got := do(t, a.env.app, "GET", "/api/tasks", "", nil).Status; got != 401 {
		t.Errorf("anonymous listing = %d, want 401", got)
	}
}
