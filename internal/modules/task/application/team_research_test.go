package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func webResearcher() application.AgentRole {
	return application.AgentRole{
		Key: "researcher", Name: "Researcher", Purpose: "Looks things up.", Tools: []string{"read_file"},
		WritesResearch: true,
	}
}

func notes(id, document string) step {
	return call(id, "report", map[string]string{"summary": "Notes are ready", "document": document})
}

func TestResearchNotesAreKeptApartFromThePRDAndReachLaterTeammates(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{webResearcher(), planner(), developer()},
		delegate("d1", "researcher", "Look up coffee shop sites"),
		notes("r1", "## Discovery\n- Orang mencari jam buka"),
		delegate("d2", "planner", "Write the PRD"),
		prdReport("r2", "# PRD\n- Tampilkan jam buka"),
		delegate("d3", "researcher", "Look up fonts"),
		notes("r3", "## Design references\n- Pakai serif hangat"),
		delegate("d4", "developer", "Build it"),
		report("r4", "Built"),
		finish("Done"),
	)
	task := r.startTeam(t, "Make a coffee shop site")
	r.run(task.ID)

	got := r.task(t, task.ID)
	if got.PRD != "# PRD\n- Tampilkan jam buka" {
		t.Fatalf("research must not replace the PRD: %q", got.PRD)
	}

	for _, want := range []string{"jam buka", "serif hangat"} {
		if !strings.Contains(got.Research, want) {
			t.Fatalf("every round of notes is kept: %q", got.Research)
		}
	}

	if r.lastEvent(task.ID, domain.EventResearch).Technical == "" {
		t.Fatal("the notes are shown in the thread")
	}

	var brief string

	for _, req := range r.model.requests {
		if text := req.Messages[0].Blocks[0].Text; strings.Contains(text, "Build it") {
			brief = text
		}
	}

	if !strings.Contains(brief, "The team's research") || !strings.Contains(brief, "serif hangat") {
		t.Fatalf("a later teammate is shown the research: %q", brief)
	}
}

func TestThePlannerWaitsUntilTheResearcherHasBeenAsked(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{webResearcher(), planner()},
		delegate("d1", "planner", "Plan the site"),
		delegate("d2", "researcher", "Look up coffee shops in Bandung"),
		notes("r1", "## Discovery\n- Orang mencari jam buka"),
		delegate("d3", "planner", "Plan the site"),
		prdReport("r2", "# PRD"),
		finish("Done"),
	)
	task := r.startTeam(t, "Make a coffee shop site")
	r.run(task.ID)

	refused := resultFor(r.model.requests[1], "d1")
	if !refused.IsError || !strings.Contains(refused.Text, "Researcher") {
		t.Fatalf("the Planner is refused until the Researcher has a job: %+v", refused)
	}

	if got := r.task(t, task.ID); got.PRD != "# PRD" || got.Research == "" {
		t.Fatalf("after the research the Planner writes the PRD: prd=%q research=%q", got.PRD, got.Research)
	}
}

func TestEveryoneAfterThePRDWaitsForOneResearchRound(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{webResearcher(), planner(), developer()},
		delegate("d1", "researcher", "Look up coffee shops"),
		notes("r1", "## Discovery\n- Orang mencari jam buka"),
		delegate("d2", "planner", "Write the PRD"),
		prdReport("r2", "# PRD"),
		delegate("d3", "developer", "Build it"),
		delegate("d4", "researcher", "Look up the stack"),
		notes("r3", "## Technical references\n- Pakai HTML biasa"),
		delegate("d5", "developer", "Build it"),
		report("r4", "Built"),
		finish("Done"),
	)
	task := r.startTeam(t, "Make a coffee shop site")
	r.run(task.ID)

	refused := resultFor(r.model.requests[5], "d3")
	if !refused.IsError || !strings.Contains(refused.Text, "Researcher") || !strings.Contains(refused.Text, "PRD") {
		t.Fatalf("the Developer is held back until a round after the PRD: %+v", refused)
	}

	if got := r.task(t, task.ID).Research; !strings.Contains(got, "Pakai HTML biasa") {
		t.Fatalf("the second round is kept too: %q", got)
	}
}

func TestAPageIsNotReportedUntilTheAppIsRunning(t *testing.T) {
	t.Parallel()

	r := newRigWith(nil, withChangingTools,
		call("c1", "edit_file", map[string]string{"path": "index.html"}),
		finish("Done"),
		call("c2", "start_site", map[string]string{"command": "python3 -m http.server"}),
		finish("Done and running"),
	)
	r.gate.skip = true
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"index.html"}}
	r.tools.results["start_site"] = application.ToolResult{Output: "started", Link: "/run/test/x/"}
	task := r.start(t, "Make a page")
	r.run(task.ID)

	refused := resultFor(r.model.requests[2], "fin")
	if !refused.IsError || !strings.Contains(refused.Text, "start_site") {
		t.Fatalf("finishing is refused while the page cannot be opened: %+v", refused)
	}

	if r.task(t, task.ID).Status != domain.StatusReview || r.lastEvent(task.ID, domain.EventSite).URL == "" {
		t.Fatalf("after the app runs the task reaches review with a link: %v", r.task(t, task.ID).Status)
	}
}
