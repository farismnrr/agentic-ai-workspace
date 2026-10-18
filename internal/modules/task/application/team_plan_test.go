package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func buildingDeveloper() application.AgentRole {
	return application.AgentRole{
		Key: "developer", Name: "Developer", Purpose: "Writes the code.", Tools: []string{"read_file", "edit_file"},
	}
}

func TestNobodyBuildsAfterThePRDUntilThePersonSaysYes(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{webResearcher(), planner(), buildingDeveloper()},
		delegate("d1", "researcher", "Look up the shop"),
		notes("r1", "## Discovery\n- Orang mencari jam buka"),
		delegate("d2", "planner", "Write the PRD"),
		prdReport("r2", "# PRD\n- Tampilkan jam buka"),
		delegate("d3", "developer", "Build it"),
		call("q1", "ask_user", map[string]any{"questions": []map[string]any{{
			"question": "Ready to build?", "type": "single",
			"options": []map[string]string{{"label": "Start building (Recommended)"}, {"label": "Change something"}},
		}}}),
		delegate("d4", "researcher", "Look up the stack"),
		notes("r3", "## Technical references\n- HTML biasa"),
		delegate("d5", "developer", "Build it"),
		call("e1", "edit_file", map[string]string{"path": "index.txt"}),
		report("r4", "Built"),
		finish("Done"),
	)
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"index.txt"}}
	r.gate.skip = true
	task := r.startPlanned(t, "Make a coffee shop site")
	r.run(task.ID)

	question := r.lastEvent(task.ID, domain.EventQuestion)
	if r.task(t, task.ID).Status != domain.StatusNeedsInput || question.Agent != "Orchestrator" {
		t.Fatalf("the lead asks the person and the task waits: %+v", question)
	}

	refused := resultFor(r.model.requests[5], "d3")
	if !refused.IsError || !strings.Contains(refused.Text, "nobody builds until the person says yes") {
		t.Fatalf("the Developer is refused before the yes: %+v", refused)
	}

	answer := application.Answer{Choice: "Start building (Recommended)"}
	if err := r.service.Answer(t.Context(), "u1", task.ID, question.ID, answer); err != nil {
		t.Fatal(err)
	}

	r.run(task.ID)

	if r.task(t, task.ID).Status != domain.StatusReview {
		t.Fatalf("after the yes the team builds and the task reaches review: %v", r.task(t, task.ID).Status)
	}
}

func TestATaskWithoutAPRDOrWithoutPlanModeIsNotHeld(t *testing.T) {
	t.Parallel()

	small := newTeamRig(nil, []application.AgentRole{buildingDeveloper()},
		delegate("d1", "developer", "Fix the typo"), report("r1", "Fixed"), finish("Done"))
	task := small.startPlanned(t, "Fix a typo")
	small.run(task.ID)

	if small.task(t, task.ID).Status == domain.StatusFailed || resultFor(small.model.requests[1], "d1").IsError {
		t.Fatal("a small change has no PRD, so it goes straight to the Developer")
	}
}

func architectRole() application.AgentRole {
	return application.AgentRole{
		Key: "architect", Name: "Architect", Purpose: "Verifies the code.", Tools: []string{"read_file"},
		ChecksArchitecture: true,
	}
}

func TestTheArchitectVerifiesTheFinalCodeBeforeTheTaskIsDone(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{architectRole(), buildingDeveloper()},
		delegate("d1", "developer", "Build the API"),
		call("e1", "edit_file", map[string]string{"path": "server/api/items.ts"}),
		report("r1", "Built"),
		finish("Done too early"),
		delegate("d2", "architect", "Verify the final code"),
		report("r2", "Sound: layers hold"),
		finish("Done"),
	)
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"server/api/items.ts"}}
	r.gate.skip = true
	task := r.startTeam(t, "Make an API")
	r.run(task.ID)

	held := resultFor(r.model.requests[4], "fin")
	if !held.IsError || !strings.Contains(held.Text, "Architect") || !strings.Contains(held.Text, "Clean Architecture") {
		t.Fatalf("the first finish waits for the Architect: %+v", held)
	}

	if got := r.task(t, task.ID).Summary; got != "Done" {
		t.Fatalf("after the Architect reported the task finishes: %q", got)
	}
}
