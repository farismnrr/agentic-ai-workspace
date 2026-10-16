package bootstrap

import (
	"os"
	"strings"
	"testing"
)

func TestAgentTeamStartsAsTheSDLCPresetsAndCanBeSavedAndReset(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t)
	cookie := jsonCookie(a.cookie)

	initial := do(t, a.env.app, "GET", "/api/agent-team", "", cookie)
	mustContain(t, initial.Body, `"key":"orchestrator"`, `"key":"business-analyst"`, `"key":"qa"`, `"lead":true`)

	saved := do(t, a.env.app, "PUT", "/api/agent-team", `{"useByDefault":true,"roles":[
		{"key":"orchestrator","name":"Boss","lead":true,"enabled":true},
		{"name":"Data Scientist","purpose":"Reads numbers","enabled":true,"access":["write","bogus-not-checked"]}]}`, cookie)
	if saved.Status != 400 {
		t.Fatalf("an unknown access group must be refused: %d %s", saved.Status, saved.Body)
	}

	saved = do(t, a.env.app, "PUT", "/api/agent-team", `{"useByDefault":true,"roles":[
		{"key":"orchestrator","name":"Boss","lead":true,"enabled":true},
		{"name":"Data Scientist","purpose":"Reads numbers","enabled":true,"access":["write"]}]}`, cookie)
	mustContain(t, saved.Body, `"name":"Boss"`, `"key":"data-scientist"`, `"useByDefault":true`)
	mustContain(t, saved.Body, `"access":["read","write"]`)

	again := do(t, a.env.app, "GET", "/api/agent-team", "", cookie)
	if !strings.Contains(string(again.Body), `"Data Scientist"`) {
		t.Fatalf("a saved team is what comes back: %s", again.Body)
	}

	reset := do(t, a.env.app, "POST", "/api/agent-team/reset", "", cookie)
	mustContain(t, reset.Body, `"key":"developer"`, `"useByDefault":false`)
}

func TestAgentTeamRefusesATeamWithoutExactlyOneLead(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t)

	for _, body := range []string{
		`{"roles":[]}`,
		`{"roles":[{"key":"a","name":"A","enabled":true},{"key":"b","name":"B","enabled":true}]}`,
		`{"roles":[{"key":"a","name":"A","lead":true},{"key":"b","name":"B","lead":true}]}`,
	} {
		if got := do(t, a.env.app, "PUT", "/api/agent-team", body, jsonCookie(a.cookie)).Status; got != 400 {
			t.Errorf("%s: want 400, got %d", body, got)
		}
	}
}

func TestTeamTaskHandsWorkToATeammateAndShowsWhoDidWhat(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t,
		toolCall("d1", "delegate", map[string]string{"role": "researcher", "brief": "Look around the project"}),
		toolCall("l1", "list_files", map[string]string{"path": "."}),
		toolCall("r1", "report", map[string]string{"summary": "The project is empty"}),
		finishCall("It is empty"),
	)

	created := do(t, a.env.app, "POST", "/api/tasks",
		`{"projectId":"`+a.project+`","prompt":"What is in here?","team":true}`, jsonCookie(a.cookie))
	if created.Status != 201 {
		t.Fatalf("task not created: %d %s", created.Status, created.Body)
	}

	mustContain(t, created.Body, `"team":true`)

	id, _ := created.JSON(t)["id"].(string)
	a.waitFor(t, id, "done")

	events := string(do(t, a.env.app, "GET", "/api/tasks/"+id+"/events", "", jsonCookie(a.cookie)).Body)
	mustContain(t, []byte(events), `"kind":"handoff"`, `"action":"assigned"`, `"action":"reported"`)
	mustContain(t, []byte(events), `"agent":"Researcher"`)

	if strings.Contains(events, `"kind":"error"`) {
		t.Fatalf("a clean team run has no errors: %s", events)
	}
}

func TestTeammateNeedsApprovalThenResumesFromItsOwnSavedConversation(t *testing.T) {
	t.Parallel()

	a := newAgentRig(t,
		toolCall("d1", "delegate", map[string]string{"role": "developer", "brief": "Make hello.txt"}),
		toolCall("w1", "write_file", map[string]string{"path": "hello.txt", "content": "hi"}),
		toolCall("r1", "report", map[string]string{"summary": "Made it"}),
		finishCall("Made hello.txt"),
	)

	created := do(t, a.env.app, "POST", "/api/tasks",
		`{"projectId":"`+a.project+`","prompt":"Make a hello file","team":true}`, jsonCookie(a.cookie))
	id, _ := created.JSON(t)["id"].(string)

	a.waitFor(t, id, "needs-input")

	var approval threadEvent

	for _, event := range a.thread(t, id) {
		if event.Kind == "approval" {
			approval = event
		}
	}

	if _, err := os.Stat(a.file("hello.txt")); approval.Agent != "Developer" || !os.IsNotExist(err) {
		t.Fatalf("the teammate asks by name before touching anything: %+v", approval)
	}

	a.waitApproving(t, id, "review")

	if data, err := os.ReadFile(a.file("hello.txt")); err != nil || string(data) != "hi" {
		t.Fatalf("after the yes the teammate's step runs: %q %v", data, err)
	}

	mustContain(t, do(t, a.env.app, "GET", "/api/tasks/"+id, "", jsonCookie(a.cookie)).Body, `"summary":"Made hello.txt"`)
}
