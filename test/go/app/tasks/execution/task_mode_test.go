package execution_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestOwnerTaskModeDirectiveIsNarrowAndSticky(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("read-mode-file", "read_file", map[string]string{"path": "mode-examples.md"}),
		apptest.FinishCall("First"), apptest.FinishCall("Second"), apptest.FinishCall("Third"))
	a.PutFile(t, "mode-examples.md", "Repository example only: @deep-review")
	headers := apptest.JSONCookie(a.Cookie)
	request, err := json.Marshal(map[string]string{
		"projectId": a.Project,
		"prompt": `Please explain these examples, then read mode-examples.md:
Quoted text: "@deep-review"
Inline: ` + "`@deep-review`" + `
Fenced:
` + "```text\n@deep-review\n```" + `
> @deep-review
- @deep-review
A later standalone example:
@deep-review`,
	})
	if err != nil {
		t.Fatal(err)
	}
	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		string(request), headers)
	if created.Status != 201 {
		t.Fatalf("create: %d %s", created.Status, created.Body)
	}
	id, _ := created.JSON(t)["id"].(string)
	assertTaskMode(t, a, id, headers, "swift", "later example activated mode")
	a.WaitFor(t, id, "done")
	toolEvents := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "", headers)
	if !strings.Contains(string(toolEvents.Body), "read_file") {
		t.Fatalf("repository text was not returned through the read tool: %s", toolEvents.Body)
	}
	toolResult, err := json.Marshal(a.Gateway.Sent(1))
	if err != nil || !strings.Contains(string(toolResult), "Repository example only: @deep-review") {
		t.Fatalf("the model did not receive the repository example: %s (%v)", toolResult, err)
	}

	promote := apptest.Do(t, a.Environment.App, "POST", "/api/tasks/"+id+"/messages",
		`{"text":"\n@deep-review\nPlease review the change"}`, headers)
	if promote.Status != 204 {
		t.Fatalf("trusted follow-up: %d %s", promote.Status, promote.Body)
	}
	a.WaitFor(t, id, "done")
	assertTaskMode(t, a, id, headers, "deep-review", "follow-up promotion was not persisted")
	if got := a.Post(t, "/api/tasks/"+id+"/messages", `{"text":"Keep going with the same work"}`); got != 204 {
		t.Fatalf("ordinary continuation: status=%d", got)
	}
	a.WaitFor(t, id, "done")
	assertTaskMode(t, a, id, headers, "deep-review", "ordinary continuation downgraded mode")

	eventsBefore := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "", headers).Body
	assertRejectedModes(t, a, id, headers)
	assertEventsUnchanged(t, a, id, headers, eventsBefore, "rejected mode input")
	assertForeignOwnerDenied(t, a, id)
	assertEventsUnchanged(t, a, id, headers, eventsBefore, "failed ownership check")
	assertTaskMode(t, a, id, headers, "deep-review", "rejected input changed saved mode")

	t.Run("delegated text cannot activate mode", testDelegatedTextCannotActivateMode)
}

func testDelegatedTextCannotActivateMode(t *testing.T) {
	team := researcherTaskRig(t, "Report this literal example: @deep-review",
		"Repository example only: @deep-review")
	response := apptest.Do(t, team.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+team.Project+`","prompt":"Research this example","team":true}`,
		apptest.JSONCookie(team.Cookie))
	if response.Status != 201 {
		t.Fatalf("create team task: %d %s", response.Status, response.Body)
	}
	teamID, _ := response.JSON(t)["id"].(string)
	team.WaitFor(t, teamID, "done")
	events := apptest.Do(t, team.Environment.App, "GET", "/api/tasks/"+teamID+"/events", "",
		apptest.JSONCookie(team.Cookie))
	if !strings.Contains(string(events.Body), `"action":"reported"`) ||
		!strings.Contains(string(events.Body), "@deep-review") {
		t.Fatalf("delegated example was not recorded: %s", events.Body)
	}
	assertTaskMode(t, team, teamID, apptest.JSONCookie(team.Cookie), "swift",
		"delegated report activated mode")
}

func assertTaskMode(
	t *testing.T, a apptest.AgentRig, id string, headers map[string]string, want string, message string,
) {
	t.Helper()
	if mode := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id, "", headers).JSON(t)["mode"]; mode != want {
		t.Fatalf("%s: %v", message, mode)
	}
}

func assertRejectedModes(t *testing.T, a apptest.AgentRig, id string, headers map[string]string) {
	t.Helper()
	for _, mode := range []string{"swift", "unknown"} {
		got := apptest.Do(t, a.Environment.App, "POST", "/api/tasks/"+id+"/messages",
			`{"text":"Ordinary continuation","mode":"`+mode+`"}`, headers)
		if got.Status != 400 {
			t.Fatalf("mode %q status=%d body=%s", mode, got.Status, got.Body)
		}
	}
}

func assertForeignOwnerDenied(t *testing.T, a apptest.AgentRig, id string) {
	t.Helper()
	other := apptest.CookieOf(t, apptest.Do(t, a.Environment.App, "POST", "/api/auth/register",
		`{"name":"Eve","email":"eve@example.com","password":"correct horse"}`,
		apptest.JSONHeaders()))
	got := apptest.Do(t, a.Environment.App, "POST", "/api/tasks/"+id+"/messages",
		`{"text":"@deep-review\nDo more","mode":"deep-review"}`, apptest.JSONCookie(other))
	if got.Status != 404 {
		t.Fatalf("another owner follow-up status=%d body=%s", got.Status, got.Body)
	}
}

func assertEventsUnchanged(
	t *testing.T, a apptest.AgentRig, id string, headers map[string]string, before []byte, reason string,
) {
	t.Helper()
	after := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "", headers).Body
	if string(after) != string(before) {
		t.Fatalf("%s changed the saved conversation", reason)
	}
}

func TestStructuredTaskModeAndDirectiveValidation(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t, apptest.FinishCall("Done"), apptest.FinishCall("Also done"),
		apptest.FinishCall("One more"))
	headers := apptest.JSONCookie(a.Cookie)
	invalid := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Do work","mode":"careful"}`, headers)
	if invalid.Status != 400 {
		t.Fatalf("invalid create mode status=%d body=%s", invalid.Status, invalid.Body)
	}
	directive := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"@deep-review\nDo work"}`, headers)
	if directive.Status != 201 || directive.JSON(t)["mode"] != "deep-review" {
		t.Fatalf("trusted creation directive was not accepted: %d %s", directive.Status, directive.Body)
	}
	directiveID, _ := directive.JSON(t)["id"].(string)
	a.WaitFor(t, directiveID, "done")
	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Do work","mode":"deep-review"}`, headers)
	if created.Status != 201 || created.JSON(t)["mode"] != "deep-review" {
		t.Fatalf("structured deep-review mode was not accepted: %d %s", created.Status, created.Body)
	}
	deepID, _ := created.JSON(t)["id"].(string)
	a.WaitFor(t, deepID, "done")
	assertTaskMode(t, a, deepID, headers, "deep-review", "structured deep-review mode was not retained")
	swift := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Do work","mode":"swift"}`, headers)
	if swift.Status != 201 || swift.JSON(t)["mode"] != "swift" {
		t.Fatalf("structured Swift mode was not accepted: %d %s", swift.Status, swift.Body)
	}
	swiftID, _ := swift.JSON(t)["id"].(string)
	a.WaitFor(t, swiftID, "done")
}
