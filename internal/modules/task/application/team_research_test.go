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
