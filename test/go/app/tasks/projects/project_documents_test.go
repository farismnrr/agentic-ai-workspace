package projects_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func TestProjectDocumentsAreExportedRepairedAndSharedWithoutThreadHistory(t *testing.T) {
	t.Parallel()
	a := apptest.NewAgentRig(t,
		apptest.ToolCall("p1", "update_plan", map[string]any{"steps": []map[string]string{
			{"title": "Researcher: investigate", "state": "pending"}, {"title": "Planner: requirements", "state": "pending"},
		}}),
		apptest.ToolCall("d0", "delegate", map[string]string{"role": "researcher", "brief": "Investigate"}),
		apptest.ToolCall("d1", "delegate", map[string]string{"role": "researcher", "brief": "Investigate"}),
		apptest.ToolCall("r1", "report", map[string]string{
			"summary": "Found evidence", "document": "# Research\nRESEARCH-SENTINEL https://example.com",
		}),
		apptest.ToolCall("d2", "delegate", map[string]string{"role": "planner", "brief": "Write requirements"}),
		apptest.ToolCall("r2", "report", map[string]string{
			"summary": "Requirements ready", "document": "# Requirements\nPRD-SENTINEL",
		}),
		apptest.FinishCall("Plan ready"), apptest.FinishCall("New task answer"),
	)
	headers := apptest.JSONCookie(a.Cookie)
	res := apptest.Do(t, a.Environment.App, "POST", "/api/tasks",
		`{"projectId":"`+a.Project+`","prompt":"Plan this project"}`, headers)
	first, _ := res.JSON(t)["id"].(string)
	a.WaitFor(t, first, "done")
	second := a.Start(t, "Read the previous project documents")
	a.WaitFor(t, second, "done")
	docs := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+second+"/documents", "", headers)
	apptest.MustContain(t, docs.Body, first+"/plan.md", first+"/requirements.md", first+"/research.md", first+"/research/")
	fileBase := "/api/projects/" + a.Project + "/file?path=project-notes/tasks/" + first
	content := apptest.Do(t, a.Environment.App, "GET", fileBase+"/research.md", "", headers)
	apptest.MustContain(t, content.Body, "RESEARCH-SENTINEL")
	// A lost export can always be rebuilt from the original task and document events.
	if err := os.RemoveAll(filepath.Join(a.File("project-notes"), "tasks", first)); err != nil {
		t.Fatal(err)
	}
	repaired := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+second+"/documents", "", headers)
	requireStatus(t, repaired, 200, "rebuild project documents")
	content = apptest.Do(t, a.Environment.App, "GET", fileBase+"/requirements.md", "", headers)
	apptest.MustContain(t, content.Body, "PRD-SENTINEL")
	a.Gateway.Mu.Lock()
	last, _ := json.Marshal(a.Gateway.Requests[len(a.Gateway.Requests)-1])
	a.Gateway.Mu.Unlock()
	if !strings.Contains(string(last), "project-notes/index.md") || strings.Contains(string(last), "RESEARCH-SENTINEL") {
		t.Fatal("a fresh task must receive the project index reference, not the previous conversation")
	}
	// Ownership is checked before document export or file access.
	other := apptest.CookieOf(t, apptest.Do(t, a.Environment.App, "POST", "/api/auth/register",
		`{"name":"Other","email":"other@example.com","password":"correct horse"}`, apptest.JSONHeaders()))
	if denied := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+second+"/documents", "",
		apptest.JSONCookie(other)); denied.Status != 404 {
		t.Fatalf("another user got documents: %d", denied.Status)
	}
}
