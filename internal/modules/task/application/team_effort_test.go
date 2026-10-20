package application_test

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type takesLevels []string

func (l takesLevels) EffortLevels(context.Context, string, string, string) []string { return l }

// leadAsks runs a team task whose lead delegates with the effort asked, and returns the effort the teammate's
// model call carried.
func leadAsks(t *testing.T, role application.AgentRole, asked string) domain.Effort {
	t.Helper()

	delegation := call("d1", "delegate", map[string]string{"role": "developer", "brief": "Fix it", "effort": asked})
	rig := newRigWith(nil, func(deps *application.Deps, _ *application.Limits) {
		lead := application.AgentRole{Key: "orchestrator", Name: "Orchestrator", Lead: true}
		deps.Team = &fakeTeam{roster: application.Roster{Lead: lead, Members: []application.AgentRole{role}}}
		deps.Scopes, deps.Delegations = deps.Transcript.(*memory), newDelegations()
		deps.Efforts = takesLevels{"low", "medium", "high"}
	}, delegation, report("r1", "Fixed"), finish("Done"))

	no, yes := false, true

	task, err := rig.service.Create(context.Background(), "u1", application.CreateInput{
		ProjectID: "proj", Prompt: "Fix it", Team: &yes, PlanFirst: &no,
	})
	if err != nil {
		t.Fatal(err)
	}

	rig.run(task.ID)

	return rig.model.requests[1].Effort
}

func TestTheLeadsEffortIsFittedToTheTeammatesModel(t *testing.T) {
	t.Parallel()

	if got := leadAsks(t, developer(), "max"); got != domain.EffortHigh {
		t.Fatalf("effort = %q, want the highest the model takes", got)
	}
}

func TestAnEffortTheRoleSetBeatsTheLeadsChoice(t *testing.T) {
	t.Parallel()

	role := developer()
	role.Effort = domain.EffortLow

	if got := leadAsks(t, role, "high"); got != domain.EffortLow {
		t.Fatalf("effort = %q, want the role's own", got)
	}
}
