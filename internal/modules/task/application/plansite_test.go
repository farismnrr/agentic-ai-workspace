package application_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// teamWith is the lead and one teammate on the person's plan, with the program that teammate runs.
func teamWith(role application.AgentRole, program *scriptedProgram, steps ...step) (*teamRig, *rig) {
	delegations := newDelegations()
	r := newRigWith(nil, func(deps *application.Deps, _ *application.Limits) {
		lead := application.AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true, Prompt: "Lead well."}
		deps.Team = &fakeTeam{roster: application.Roster{Lead: lead, Members: []application.AgentRole{role}}}
		deps.Scopes, deps.Delegations = deps.Transcript.(*memory), delegations
		deps.CLI, deps.Sessions = program, sessionBook{}
	}, steps...)

	return &teamRig{rig: r, delegations: delegations}, r
}

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
	r, rig := teamWith(ops, program, delegate("d1", "devops", "Start the app"), finish("Done"))
	rig.tools.extra = []domain.ToolDef{{Name: "start_site"}, {Name: "stop_site"}}
	rig.tools.results["start_site"] = application.ToolResult{Output: "The app is running.", Link: "/run/p/"}
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

// A teammate on a plan who may run commands can ask for a website through the server's tool, and the program is told
// which task it works for (the server's own record), so its container can be given that task's socket.
func TestARoleThatRunsCommandsCanAskForAWebsiteOnAPlanAndTheRequestNamesTheTask(t *testing.T) {
	t.Parallel()

	program := &scriptedProgram{
		say:   "Asked.",
		calls: []scriptedCall{{name: "request_egress", input: map[string]any{"host": "a.example.com", "reason": "x"}}},
	}
	dev := application.AgentRole{
		Key: "developer", Name: "Developer", Purpose: "Builds.", ProviderID: "plan:claude", Model: "sonnet",
		Tools: []string{"run_command", "edit_file"},
	}
	r, rig := teamWith(dev, program, delegate("d1", "developer", "Build it"), finish("Done"))
	rig.tools.extra = []domain.ToolDef{{Name: "request_egress"}}
	rig.tools.results["request_egress"] = application.ToolResult{Output: "Asked the person."}
	task := r.startTeam(t, "Build it")
	r.run(task.ID)

	if len(program.requests) != 1 || program.requests[0].TaskID != task.ID ||
		!slices.Equal(toolsOf(program.requests[0]), []string{"request_egress"}) {
		t.Fatalf("the program must be told its task and get the tool: %+v", program.requests)
	}

	if !slices.Contains(rig.tools.executed, "request_egress") {
		t.Fatalf("executed=%v", rig.tools.executed)
	}
}

func TestARoleThatCannotRunCommandsGetsNoWayToAskForAWebsite(t *testing.T) {
	t.Parallel()

	program := &scriptedProgram{say: "Read it."}
	reader := application.AgentRole{
		Key: "reviewer", Name: "Reviewer", Purpose: "Reads.", ProviderID: "plan:claude", Model: "sonnet",
		Tools: []string{"read_file"},
	}
	r, rig := teamWith(reader, program, delegate("d1", "reviewer", "Read it"), finish("Done"))
	rig.tools.extra = []domain.ToolDef{{Name: "request_egress"}}
	r.run(r.startTeam(t, "Read it").ID)

	if len(program.requests) != 1 || program.requests[0].Bridge != nil {
		t.Fatalf("no server tools expected: %+v", program.requests)
	}
}
