package execution_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestAgentTeamStartsAsTheSDLCPresetsAndCanBeSavedAndReset(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t)
	cookie := apptest.JSONCookie(a.Cookie)

	initial := apptest.Do(t, a.Environment.App, "GET", "/api/agent-team", "", cookie)
	apptest.MustContain(t, initial.Body, `"key":"orchestrator"`, `"key":"planner"`, `"key":"designer"`,
		`"key":"business-analyst"`)
	apptest.MustContain(t, initial.Body, `"key":"qa"`, `"lead":true`)

	saved := apptest.Do(t, a.Environment.App, "PUT", "/api/agent-team", `{"useByDefault":true,"roles":[
		{"key":"orchestrator","name":"Boss","lead":true,"enabled":true},
		{"name":"Data Scientist","purpose":"Reads numbers","enabled":true,"access":["write","bogus-not-checked"]}]}`, cookie)
	if saved.Status != 400 {
		t.Fatalf("an unknown access group must be refused: %d %s", saved.Status, saved.Body)
	}

	saved = apptest.Do(t, a.Environment.App, "PUT", "/api/agent-team", `{"useByDefault":true,"roles":[
		{"key":"orchestrator","name":"Boss","lead":true,"enabled":true},
		{"name":"Data Scientist","purpose":"Reads numbers","enabled":true,"access":["write"]}]}`, cookie)
	apptest.MustContain(t, saved.Body, `"name":"Boss"`, `"key":"data-scientist"`, `"useByDefault":true`)
	apptest.MustContain(t, saved.Body, `"access":["read","write"]`)

	again := apptest.Do(t, a.Environment.App, "GET", "/api/agent-team", "", cookie)
	if !strings.Contains(string(again.Body), `"Data Scientist"`) {
		t.Fatalf("a saved team is what comes back: %s", again.Body)
	}

	reset := apptest.Do(t, a.Environment.App, "POST", "/api/agent-team/reset", "", cookie)
	apptest.MustContain(t, reset.Body, `"key":"developer"`, `"useByDefault":false`)
}

func TestAgentTeamRefusesAnEmptyOrOversizedTeam(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t)

	for _, body := range []string{
		`{"roles":[]}`,
		`{"roles":[{"key":"orchestrator","name":"A"},{"key":"b","name":"B","purpose":"` + strings.Repeat("x", 300) + `"}]}`,
	} {
		if got := apptest.Do(t, a.Environment.App, "PUT", "/api/agent-team", body,
			apptest.JSONCookie(a.Cookie)).Status; got != 400 {
			t.Errorf("%s: want 400, got %d", body, got)
		}
	}
}

func TestTeamTaskHandsWorkToATeammateAndShowsWhoDidWhat(t *testing.T) {
	t.Parallel()

	a := researcherTaskRig(t, "Look around the project", "The project is empty")

	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"@deep-review\nWhat is in here?","team":true}`,
		apptest.JSONCookie(a.Cookie))
	if created.Status != 201 {
		t.Fatalf("task not created: %d %s", created.Status, created.Body)
	}
	apptest.MustContain(t, created.Body, `"mode":"deep-review"`)

	id, _ := created.JSON(t)["id"].(string)
	a.WaitFor(t, id, "done")
	a.Gateway.Mu.Lock()
	requests, _ := json.Marshal(a.Gateway.Requests)
	a.Gateway.Mu.Unlock()
	if strings.Count(string(requests), "Saved task verification mode: deep-review.") < 2 {
		t.Fatalf("the saved mode must reach the lead and delegated teammate: %s", requests)
	}
	events := string(apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/events", "",
		apptest.JSONCookie(a.Cookie)).Body)
	apptest.MustContain(t, []byte(events), `"kind":"handoff"`, `"action":"assigned"`, `"action":"reported"`)
	apptest.MustContain(t, []byte(events), `"agent":"Researcher"`)
	if strings.Contains(events, `"kind":"error"`) {
		t.Fatalf("a clean team run has no errors: %s", events)
	}
}

func researcherTaskRig(t *testing.T, brief, report string) apptest.AgentRig {
	t.Helper()
	return apptest.NewAgentRig(t,
		apptest.ToolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Researcher: look around the project", "state": "pending"},
		}}),
		apptest.ToolCall("d0", "delegate", map[string]string{"role": "researcher", "brief": brief}),
		apptest.ToolCall("d1", "delegate", map[string]string{"role": "researcher", "brief": brief}),
		apptest.ToolCall("l1", "list_files", map[string]string{"path": "."}),
		apptest.ToolCall("r1", "report", map[string]string{"summary": report}),
		apptest.FinishCall("It is empty"),
	)
}

func TestTeammateNeedsApprovalThenResumesFromItsOwnSavedConversation(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t,
		apptest.ToolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Developer: make hello.txt", "state": "pending"},
		}}),
		apptest.ToolCall("d0", "delegate", map[string]string{"role": "developer", "brief": "Make hello.txt"}),
		apptest.ToolCall("d1", "delegate", map[string]string{"role": "developer", "brief": "Make hello.txt"}),
		apptest.ToolCall("w1", "write_file", map[string]string{"path": "hello.txt", "content": "hi"}),
		apptest.ToolCall("r1", "report", map[string]string{"summary": "Made it"}),
		apptest.FinishCall("Made hello.txt"), // held: the work is not saved in git yet
		apptest.ToolCall("d2", "delegate", map[string]string{"role": "devops", "brief": "Commit and push"}),
		// the test sandbox has no bash, so the commit cannot succeed: the hold gives up after a few tries
		apptest.ToolCall("c1", "run_command", map[string]string{"command": "ship \"Add hello\""}),
		apptest.ToolCall("r2", "report", map[string]string{"summary": "Could not save"}),
		apptest.FinishCall("Made hello.txt"),
		apptest.FinishCall("Made hello.txt"),
		apptest.FinishCall("Made hello.txt"),
		apptest.FinishCall("Made hello.txt"), // the run restarted after the approval of the command, so it counts again
	)

	created := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Make a hello file","team":true}`, apptest.JSONCookie(a.Cookie))
	id, _ := created.JSON(t)["id"].(string)

	a.WaitFor(t, id, "needs-input")

	var approval apptest.ThreadEvent

	for _, event := range a.Thread(t, id) {
		if event.Kind == "approval" {
			approval = event
		}
	}

	if _, err := os.Stat(a.File("hello.txt")); approval.Agent != "Developer" || !os.IsNotExist(err) {
		t.Fatalf("the teammate asks by name before touching anything: %+v", approval)
	}

	a.WaitApproving(t, id, "review")

	if data, err := os.ReadFile(a.File("hello.txt")); err != nil || string(data) != "hi" {
		t.Fatalf("after the yes the teammate's step runs: %q %v", data, err)
	}

	apptest.MustContain(t, apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id, "",
		apptest.JSONCookie(a.Cookie)).Body, `"summary":"Made hello.txt"`)
}

func TestBuiltInRolesStayWhateverIsSaved(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t)
	cookie := apptest.JSONCookie(a.Cookie)

	saved := apptest.Do(t, a.Environment.App, "PUT", "/api/agent-team",
		`{"roles":[{"key":"orchestrator","name":"Boss","lead":true},{"name":"Mine","enabled":true}]}`, cookie)
	apptest.MustContain(t, saved.Body, `"key":"planner"`, `"key":"developer"`, `"key":"qa"`, `"key":"mine"`)

	// Another person's team is their own: what one person saves never reaches another.
	second := apptest.CookieOf(t, apptest.Do(t, a.Environment.App, "POST", "/api/auth/register",
		`{"name":"Budi","email":"budi@example.com","password":"correct horse"}`, apptest.JSONHeaders()))
	theirs := apptest.Do(t, a.Environment.App, "GET", "/api/agent-team", "", apptest.JSONCookie(second))

	apptest.MustContain(t, theirs.Body, `"name":"Orchestrator"`, `"key":"planner"`)

	if strings.Contains(string(theirs.Body), `"key":"mine"`) {
		t.Fatalf("a role one person added must not appear for another: %s", theirs.Body)
	}
}
