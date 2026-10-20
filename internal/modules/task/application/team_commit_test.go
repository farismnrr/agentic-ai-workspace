package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestTheWorkIsSavedInGitBeforeTheTeamReports(t *testing.T) {
	t.Parallel()

	r := newTeamRig(moreSteps, []application.AgentRole{appBuilder()},
		delegate("d1", "developer", "Build the page"),
		call("e1", "edit_file", map[string]string{"path": "app/page.ts"}),
		report("r1", "Built"),
		finish("Done too early"),
		delegate("d2", "developer", "Commit and push"),
		call("c1", "run_command", map[string]string{"command": "ship \"Add the page\""}),
		report("r2", "Saved"),
		finish("Done"),
	)
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"app/page.ts"}}
	shipped := &domain.RunInfo{Name: "ship \"Add the page\""}
	r.tools.results["run_command"] = application.ToolResult{Output: "ok", Run: shipped}
	r.gate.skip = true
	task := r.startTeam(t, "Make a page")
	r.run(task.ID)

	held := resultFor(r.model.requests[4], "fin")
	if !held.IsError || !strings.Contains(held.Text, "ship") {
		t.Fatalf("the first finish waits until the work is saved: %+v", held)
	}

	if got := r.task(t, task.ID).Summary; got != "Done" {
		t.Fatalf("after the commit the team reports: %q", got)
	}
}

func TestAChangeAfterTheCommitIsSavedAgain(t *testing.T) {
	t.Parallel()

	events := []domain.Event{
		{Seq: 1, Kind: domain.EventAction, Files: []string{"a.txt"}},
		{Seq: 2, Kind: domain.EventAction, Run: &domain.RunInfo{Name: "ship \"Add a\""}},
		{Seq: 3, Kind: domain.EventAction, Files: []string{"b.txt"}},
	}

	if !application.ChangedSinceCommit(events) {
		t.Fatal("a file changed after the commit must be saved again")
	}

	if application.ChangedSinceCommit(events[:2]) {
		t.Fatal("nothing changed after the commit")
	}

	failed := []domain.Event{
		{Seq: 1, Kind: domain.EventAction, Files: []string{"a.txt"}},
		{Seq: 2, Kind: domain.EventAction, Run: &domain.RunInfo{Name: "ship \"x\"", ExitCode: 4}},
	}
	if !application.ChangedSinceCommit(failed) {
		t.Fatal("a commit that failed saved nothing")
	}
}
