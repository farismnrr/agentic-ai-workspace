package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func extensionBuilder() application.AgentRole {
	return application.AgentRole{
		Key: "devops", Name: "DevOps", Purpose: "Ships.",
		Tools: []string{"read_file", "edit_file", "deploy_extension"},
	}
}

func TestAnExtensionIsPackagedBeforeTheTeamReports(t *testing.T) {
	t.Parallel()

	r := newTeamRig(moreSteps, []application.AgentRole{extensionBuilder()},
		delegate("d1", "devops", "Build the extension"),
		call("e1", "edit_file", map[string]string{"path": "entrypoints/popup/main.ts"}),
		report("r1", "Built"),
		finish("Done too early"),
		delegate("d2", "devops", "Package it"),
		call("x1", "deploy_extension", map[string]string{"path": ".output/chrome-mv3"}),
		report("r2", "Packaged"),
		finish("Done"),
	)
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"entrypoints/popup/main.ts"}}
	r.tools.results["deploy_extension"] = application.ToolResult{Output: "packaged", ExtensionLink: "/app/extensions/p"}
	r.gate.skip = true
	task := r.startTeam(t, "Make an extension")
	r.run(task.ID)

	held := resultFor(r.model.requests[4], "fin")
	if !held.IsError || !strings.Contains(held.Text, "deploy_extension") {
		t.Fatalf("the first finish waits until the extension is packaged: %+v", held)
	}

	if r.lastEvent(task.ID, domain.EventExtension).URL != "/app/extensions/p" {
		t.Fatal("the thread links to where the person installs it")
	}

	if got := r.task(t, task.ID).Summary; got != "Done" {
		t.Fatalf("after it is packaged the team reports: %q", got)
	}
}
