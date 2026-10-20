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
		Tools: []string{"read_file", "edit_file", "run_command", "start_site"},
	}
}

func moreSteps(lim *application.Limits) { lim.MaxSteps, lim.TeamSteps = 40, 40 }

func TestAnAppStartsFreelyWhileBuiltAndTheSecurityReviewComesOnceAtTheEnd(t *testing.T) {
	t.Parallel()

	r := newTeamRig(moreSteps, []application.AgentRole{securityRole(), appBuilder()},
		delegate("d1", "developer", "Build the API and start it"),
		call("e1", "edit_file", map[string]string{"path": "server/api/items.ts"}),
		call("s1", "start_site", map[string]string{"command": "node .output/server/index.mjs", "port": "3000"}),
		report("r1", "Built and started"),
		finish("Done too early"),
		delegate("d2", "security", "Check the final code"),
		report("r2", "Safe to start"),
		delegate("d3", "developer", "Save the work"),
		call("c1", "run_command", map[string]string{"command": "ship \"Add the API\""}),
		report("r3", "Saved"),
		finish("Done"),
	)
	shipped := &domain.RunInfo{Name: "ship \"Add the API\""}
	r.tools.results["run_command"] = application.ToolResult{Output: "ok", Run: shipped}
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"server/api/items.ts"}}
	r.tools.results["start_site"] = application.ToolResult{Output: "started", Link: "/run/x/y/"}
	r.gate.skip = true
	task := r.startTeam(t, "Make an app")
	r.run(task.ID)

	if started := resultFor(r.model.requests[3], "s1"); started.IsError {
		t.Fatalf("the app starts while it is being built, without a review: %+v", started)
	}

	held := resultFor(r.model.requests[5], "fin")
	if !held.IsError || !strings.Contains(held.Text, "Security") {
		t.Fatalf("the first finish waits for the Security check: %+v", held)
	}

	if got := r.task(t, task.ID).Summary; got != "Done" {
		t.Fatalf("after the Security check the task finishes: %q", got)
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
