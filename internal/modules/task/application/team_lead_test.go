package application_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// changing is a toolbox that can look (read_file) and change things (edit_file).
type changing struct{ *fakeTools }

func (changing) Definitions() []domain.ToolDef {
	return []domain.ToolDef{{Name: "read_file"}, {Name: "edit_file"}}
}

// newChangingRig is a team rig whose tools include one that changes files, with and without a team.
func newChangingRig(members []application.AgentRole, steps ...step) *teamRig {
	r := newTeamRig(nil, members, steps...)
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"a.html"}}
	r.gate.skip = true

	return r
}

func withChangingTools(deps *application.Deps, _ *application.Limits) {
	deps.Tools = changing{deps.Tools.(*fakeTools)}
}

func TestTheLeadNeverChangesAnythingItselfButTeammatesDo(t *testing.T) {
	t.Parallel()

	delegations := newDelegations()
	rig := newRigWith(nil, func(deps *application.Deps, lim *application.Limits) {
		withChangingTools(deps, lim)
		deps.Team = &fakeTeam{roster: application.Roster{
			Lead:    application.AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true},
			Members: []application.AgentRole{developer()},
		}}
		deps.Scopes, deps.Delegations = deps.Transcript.(*memory), delegations
	},
		call("c1", "edit_file", map[string]string{"path": "a.html"}),
		delegate("d1", "developer", "Fix the typo"),
		call("c2", "edit_file", map[string]string{"path": "a.html"}),
		report("r1", "Fixed"),
		finish("Done"),
	)
	rig.gate.skip = true
	rig.tools.risks["edit_file"] = domain.RiskLow
	rig.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"a.html"}}
	r := &teamRig{rig: rig, delegations: delegations}
	task := r.startTeam(t, "Fix the typo")
	r.run(task.ID)

	if lead := toolNames(r.model.requests[0]); slices.Contains(lead, "edit_file") || !slices.Contains(lead, "read_file") {
		t.Fatalf("the lead is offered tools that look, not tools that change: %v", lead)
	}

	if mate := toolNames(r.model.requests[2]); !slices.Contains(mate, "edit_file") {
		t.Fatalf("a teammate who may change files is offered the tool: %v", mate)
	}

	refused := resultFor(r.model.requests[1], "c1")
	if !refused.IsError || !strings.Contains(refused.Text, "never change files") {
		t.Fatalf("a change the lead tries anyway is refused and says why: %+v", refused)
	}

	if !slices.Equal(r.tools.executed, []string{"edit_file"}) {
		t.Fatalf("only the teammate's change ran, once: %v", r.tools.executed)
	}

	events, _ := r.mem.EventsAfter(context.Background(), task.ID, 0)
	if steps, _ := stepsBy(events, "Developer"); steps != 1 {
		t.Fatalf("the change is the Developer's: %d", steps)
	}
}

func TestSoloTasksStillChangeThingsThemselves(t *testing.T) {
	t.Parallel()

	r := newRigWith(nil, withChangingTools, call("c1", "edit_file", map[string]string{"path": "a.html"}), finish("Done"))
	r.gate.skip = true
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"a.html"}}
	task := r.start(t, "Fix the typo")
	r.run(task.ID)

	if offered := toolNames(r.model.requests[0]); !slices.Contains(offered, "edit_file") || len(r.tools.executed) != 1 {
		t.Fatalf("without a team the agent changes things as before: %v %v", offered, r.tools.executed)
	}
}

func TestTheLeadSaysSoWhenNoOneCanChangeFiles(t *testing.T) {
	t.Parallel()

	r := newChangingRig([]application.AgentRole{researcher()}, finish("Nobody can do it"))
	task := r.startTeam(t, "Fix the typo")
	r.run(task.ID)

	if lead := r.model.requests[0].System; !strings.Contains(lead, "No one on your team can change files") ||
		!strings.Contains(lead, "never change anything yourself") {
		t.Fatalf("the lead is told to ask for a Developer, not to do it itself: %q", lead)
	}
}
