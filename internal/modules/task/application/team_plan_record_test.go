package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func requirePlan(lim *application.Limits) { lim.RequirePlan = true }

func planStep(title, state string) map[string]string {
	return map[string]string{"title": title, "state": state}
}

func TestARequestIsInThePlanBeforeAnyTeammateStarts(t *testing.T) {
	t.Parallel()

	r := newTeamRig(requirePlan, []application.AgentRole{buildingDeveloper()},
		delegate("d1", "developer", "Fix the title"),
		call("p1", "update_plan", map[string]any{"steps": []map[string]string{
			planStep("Developer: fix the title", "pending"),
		}}),
		delegate("d2", "developer", "Fix the title"),
		report("r1", "Fixed"),
		finish("Done"),
	)
	r.gate.skip = true
	task := r.startTeam(t, "Fix the title")
	r.run(task.ID)

	held := resultFor(r.model.requests[1], "d1")
	if !held.IsError || !strings.Contains(held.Text, "update_plan") {
		t.Fatalf("the job waits until the request is in the plan: %+v", held)
	}

	if !slicesContain(r.kinds(task.ID), domain.EventPlan) {
		t.Fatalf("the thread shows the plan being set: %v", r.kinds(task.ID))
	}

	got := r.task(t, task.ID).Plan
	if len(got) != 1 || got[0].State != domain.PlanDone {
		t.Fatalf("the step follows its teammate to done: %+v", got)
	}
}

func TestATeammateThatIsStuckLeavesAFailedStepWithTheReason(t *testing.T) {
	t.Parallel()

	r := newTeamRig(requirePlan, []application.AgentRole{buildingDeveloper()},
		call("p1", "update_plan", map[string]any{"steps": []map[string]string{
			planStep("Developer: build the page", "pending"),
		}}),
		delegate("d1", "developer", "Build the page"),
		report("r1", "Stuck: the build says out of memory twice"),
		finish("Could not finish"),
	)
	r.gate.skip = true
	task := r.startTeam(t, "Build the page")
	r.run(task.ID)

	got := r.task(t, task.ID).Plan
	if len(got) != 1 || got[0].State != domain.PlanFailed || !strings.Contains(got[0].Detail, "out of memory") {
		t.Fatalf("a stuck teammate must show on the todo list: %+v", got)
	}
}

func slicesContain(kinds []domain.EventKind, want domain.EventKind) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}

	return false
}
