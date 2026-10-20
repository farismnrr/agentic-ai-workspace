package application_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestADevOpsOnAPlanStartsTheAppThroughTheServersTool(t *testing.T) {
	t.Parallel()

	program := &scriptedProgram{
		say:   "The app is in Sites.",
		calls: []scriptedCall{{name: "start_site", input: map[string]any{"command": "x", "port": 3000}}},
	}
	ops := application.AgentRole{
		Key: "devops", Name: "DevOps", Purpose: "Runs the app.", ProviderID: "plan:claude", Model: "sonnet",
		Tools: []string{"run_command", "start_site", "stop_site"},
	}
	delegations := newDelegations()
	rig := newRigWith(nil, func(deps *application.Deps, _ *application.Limits) {
		lead := application.AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true, Prompt: "Lead well."}
		deps.Team = &fakeTeam{roster: application.Roster{Lead: lead, Members: []application.AgentRole{ops}}}
		deps.Scopes, deps.Delegations = deps.Transcript.(*memory), delegations
		deps.CLI, deps.Sessions = program, sessionBook{}
	}, delegate("d1", "devops", "Start the app"), finish("Done"))
	rig.tools.extra = []domain.ToolDef{{Name: "start_site"}, {Name: "stop_site"}}
	rig.tools.results["start_site"] = application.ToolResult{Output: "The app is running.", Link: "/run/p/"}
	r := &teamRig{rig: rig, delegations: delegations}
	r.run(r.startTeam(t, "Run the app").ID)

	if len(program.requests) != 1 || program.requests[0].Bridge == nil ||
		!slices.Equal(toolsOf(program.requests[0]), []string{"start_site", "stop_site"}) {
		t.Fatalf("the program must get the site tools: %+v", program.requests)
	}

	if !slices.Contains(rig.tools.executed, "start_site") || len(program.answers) != 1 ||
		!strings.Contains(program.answers[0], "running") {
		t.Fatalf("executed=%v answers=%v", rig.tools.executed, program.answers)
	}
}
