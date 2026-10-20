package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestTheHandoffCardShowsTheEffortTheOrchestratorAskedFor(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{buildingDeveloper()},
		call("d1", "delegate", map[string]string{"role": "developer", "brief": "Fix the title", "effort": "medium"}),
		report("r1", "Fixed"),
		finish("Done"),
	)
	r.gate.skip = true
	task := r.startTeam(t, "Fix the title")
	r.run(task.ID)

	var assigned string

	for _, event := range r.mem.events[task.ID] {
		if event.Kind == domain.EventHandoff && strings.Contains(event.Text, "take this on") {
			assigned = event.Effort
		}
	}

	if assigned != "medium" {
		t.Fatalf("the card shows the effort that was asked for, never auto: %q", assigned)
	}
}
