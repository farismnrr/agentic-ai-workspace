package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectDocumentsAreExportedRepairedAndSharedWithoutThreadHistory(t *testing.T) {
	t.Parallel()
	a := newAgentRig(t,
		toolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Researcher: investigate", "state": "pending"}, {"title": "Planner: requirements", "state": "pending"},
		}}),
		toolCall("d0", "delegate", map[string]string{"role": "researcher", "brief": "Investigate"}),
		toolCall("d1", "delegate", map[string]string{"role": "researcher", "brief": "Investigate"}),
		toolCall("r1", "report", map[string]string{
			"summary": "Found evidence", "document": "# Research\nRESEARCH-SENTINEL https://example.com",
		}),
		toolCall("d2", "delegate", map[string]string{"role": "planner", "brief": "Write requirements"}),
		toolCall("r2", "report", map[string]string{
			"summary": "Requirements ready", "document": "# Requirements\nPRD-SENTINEL",
		}),
		finishCall("Plan ready"), finishCall("New task answer"),
	)
	headers := jsonCookie(a.cookie)
	input, _ := json.Marshal(map[string]any{"projectId": a.project, "prompt": "Plan this project", "team": true})
	res := do(t, a.env.app, "POST", "/api/tasks", string(input), headers)
	first, _ := res.JSON(t)["id"].(string)
	a.waitFor(t, first, "done")
	second := a.start(t, "Read the previous project documents")
	a.waitFor(t, second, "done")
	docs := do(t, a.env.app, "GET", "/api/tasks/"+second+"/documents", "", headers)
	mustContain(t, docs.Body, first+"/plan.md", first+"/requirements.md", first+"/research.md", first+"/research/")
	fileBase := "/api/projects/" + a.project + "/file?path=project-notes/tasks/" + first
	content := do(t, a.env.app, "GET", fileBase+"/research.md", "", headers)
	mustContain(t, content.Body, "RESEARCH-SENTINEL")
	// A lost export can always be rebuilt from the original task and document events.
	if err := os.RemoveAll(filepath.Join(a.file("project-notes"), "tasks", first)); err != nil {
		t.Fatal(err)
	}
	repaired := do(t, a.env.app, "GET", "/api/tasks/"+second+"/documents", "", headers)
	if repaired.Status != 200 {
		t.Fatal(string(repaired.Body))
	}
	content = do(t, a.env.app, "GET", fileBase+"/requirements.md", "", headers)
	mustContain(t, content.Body, "PRD-SENTINEL")
	a.gateway.mu.Lock()
	last, _ := json.Marshal(a.gateway.requests[len(a.gateway.requests)-1])
	a.gateway.mu.Unlock()
	if !strings.Contains(string(last), "project-notes/index.md") || strings.Contains(string(last), "RESEARCH-SENTINEL") {
		t.Fatal("a fresh task must receive the project index reference, not the previous conversation")
	}
	// Ownership is checked before document export or file access.
	other := cookieOf(t, do(t, a.env.app, "POST", "/api/auth/register",
		`{"name":"Other","email":"other@example.com","password":"correct horse"}`, jsonHeaders()))
	if denied := do(t, a.env.app, "GET", "/api/tasks/"+second+"/documents", "", jsonCookie(other)); denied.Status != 404 {
		t.Fatalf("another user got documents: %d", denied.Status)
	}
}
