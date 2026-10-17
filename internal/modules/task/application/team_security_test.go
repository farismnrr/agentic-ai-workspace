package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func securityRole() application.AgentRole {
	return application.AgentRole{
		Key: "security", Name: "Security", Purpose: "Checks the code.", Tools: []string{"read_file", "run_command"},
		ChecksSecurity: true,
	}
}

func appBuilder() application.AgentRole {
	return application.AgentRole{
		Key: "developer", Name: "Developer", Purpose: "Writes the code.",
		Tools: []string{"read_file", "edit_file", "start_site"},
	}
}

func TestAnAppWithCodeStartsOnlyAfterTheSecurityReview(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{securityRole(), appBuilder()},
		delegate("d1", "developer", "Build the API and start it"),
		call("e1", "edit_file", map[string]string{"path": "server/api/items.ts"}),
		call("s1", "start_site", map[string]string{"command": "node .output/server/index.mjs", "port": "3000"}),
		report("r1", "Built, not started"),
		delegate("d2", "security", "Check the app"),
		report("r2", "Safe to start"),
		delegate("d3", "developer", "Start it"),
		call("s2", "start_site", map[string]string{"command": "node .output/server/index.mjs", "port": "3000"}),
		report("r3", "Started"),
		finish("Done"),
	)
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"server/api/items.ts"}}
	r.tools.results["start_site"] = application.ToolResult{Output: "started", Link: "/run/x/y/"}
	r.gate.skip = true
	task := r.startTeam(t, "Make an app")
	r.run(task.ID)

	held := resultFor(r.model.requests[3], "s1")
	if !held.IsError || !strings.Contains(held.Text, "Security") {
		t.Fatalf("the start is refused before the review: %+v", held)
	}

	if r.lastEvent(task.ID, domain.EventSite).URL == "" {
		t.Fatal("after the review the app starts and the person gets a link")
	}
}

func TestAPlainPageNeedsNoSecurityReview(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{securityRole(), appBuilder()},
		delegate("d1", "developer", "Make a page and start it"),
		call("e1", "edit_file", map[string]string{"path": "index.html"}),
		call("s1", "start_site", map[string]string{"command": "python3 -m http.server 8000", "port": "8000"}),
		report("r1", "Started"),
		finish("Done"),
	)
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"index.html"}}
	r.tools.results["start_site"] = application.ToolResult{Output: "started", Link: "/run/x/y/"}
	r.gate.skip = true
	task := r.startTeam(t, "Make a page")
	r.run(task.ID)

	if r.lastEvent(task.ID, domain.EventSite).URL == "" {
		t.Fatal("static pages with no code start without a review")
	}
}
