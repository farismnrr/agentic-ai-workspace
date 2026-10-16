package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestTeammateIsShownTheTeamsPlan(t *testing.T) {
	t.Parallel()

	plan := map[string]any{"steps": []map[string]string{
		{"title": "Fix the button colour", "state": "active"}, {"title": "Check the page", "state": "pending"},
	}}
	r := newTeamRig(nil, []application.AgentRole{developer()},
		call("p1", "update_plan", plan),
		delegate("d1", "developer", "Fix the button"),
		report("r1", "Fixed"),
		finish("Done"),
	)
	task := r.startTeam(t, "Fix the button")
	r.run(task.ID)

	brief := r.model.requests[2].Messages[0].Blocks[0].Text
	if !strings.Contains(brief, "Fix the button") || !strings.Contains(brief, "The team's plan so far") ||
		!strings.Contains(brief, "[active] Fix the button colour") {
		t.Fatalf("a teammate starts from the brief and the shared plan: %q", brief)
	}
}

func TestTeamIsStoppedOnlyWhenJobsStopGettingAnywhere(t *testing.T) {
	t.Parallel()

	r := newTeamRig(func(l *application.Limits) { l.MaxIdleJobs = 2 }, []application.AgentRole{developer()},
		delegate("d1", "developer", "One"), report("r1", "Done one"),
		delegate("d2", "developer", "Two"), report("r2", "Done two"),
		delegate("d3", "developer", "Three"),
		finish("Told the person what is stuck"),
	)
	task := r.startTeam(t, "Do many things")
	r.run(task.ID)

	third := resultFor(r.model.requests[5], "d3")
	if !third.IsError || !strings.Contains(third.Text, "last 2 jobs") || len(r.model.requests) != 6 {
		t.Fatalf("a third job that would get nowhere is refused and no teammate starts: %+v requests=%d",
			third, len(r.model.requests))
	}
}

func TestAnyNumberOfJobsIsFineWhileTheyGetSomewhere(t *testing.T) {
	t.Parallel()

	var script []step

	for _, id := range []string{"1", "2", "3", "4", "5"} {
		script = append(script,
			delegate("d"+id, "developer", "Module "+id),
			call("c"+id, "edit_file", map[string]string{"path": "m" + id + ".py"}),
			report("r"+id, "Built module "+id))
	}

	script = append(script, finish("All five modules are built"))
	r := newTeamRig(func(l *application.Limits) { l.MaxIdleJobs = 2; l.TeamSteps = 60 },
		[]application.AgentRole{developer()}, script...)
	r.gate.skip = true
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"m.py"}}
	task := r.startTeam(t, "Build five modules")
	r.run(task.ID)

	if got := r.task(t, task.ID); got.Status != domain.StatusReview || len(r.tools.executed) != 5 {
		t.Fatalf("a team that keeps making progress is never cut off for its number of jobs: %v %d",
			got.Status, len(r.tools.executed))
	}
}

func TestLeadSizesTheRequestBeforeUsingTheTeam(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()},
		delegate("d1", "developer", "Work"), report("r1", "Done"), finish("Done"))
	task := r.startTeam(t, "Do it")
	r.run(task.ID)

	lead, mate := r.model.requests[0].System, r.model.requests[1].System

	wanted := []string{
		"Size the request first", "Small:", "Medium:", "Large:", "pick the smaller size", "two rounds", "never Small",
		"is not done until a check ran",
	}
	for _, want := range wanted {
		if !strings.Contains(lead, want) {
			t.Errorf("the lead should be told %q", want)
		}
	}

	if strings.Contains(mate, "Size the request first") {
		t.Error("a teammate does not size or delegate")
	}
}

func TestSmallTaskNeedsNoPlanToReachReview(t *testing.T) {
	t.Parallel()

	r := newRig(nil, call("c1", "edit_file", map[string]string{"path": "a.html"}), finish("Fixed the typo"))
	r.gate.skip = true
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"a.html"}}
	task := r.start(t, "Fix the typo")
	r.run(task.ID)

	if got := r.task(t, task.ID); got.Status != domain.StatusReview || len(got.Plan) != 0 {
		t.Fatalf("a small job needs no plan and still reaches review: %v %v", got.Status, got.Plan)
	}
}

func TestEveryStepOfATeamTaskNamesWhoDidIt(t *testing.T) {
	t.Parallel()

	qa := application.AgentRole{Key: "qa", Name: "QA", ProviderID: "prov", Model: "qa-model", Tools: []string{"run_task"}}
	r := newTeamRig(nil, []application.AgentRole{developer(), qa},
		delegate("d1", "developer", "Change it"),
		call("c1", "edit_file", map[string]string{"path": "a.html"}),
		report("r1", "Changed"),
		delegate("d2", "qa", "Run the tests"),
		call("c2", "run_task", map[string]string{"name": "test"}),
		report("r2", "Passed"),
		finish("Done"),
	)
	r.gate.skip = true
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"a.html"}}
	r.tools.results["run_task"] = application.ToolResult{Output: "ok", Run: &domain.RunInfo{Name: "test"}}
	task := r.startTeam(t, "Change and test")
	r.run(task.ID)

	who := stepOwners(t, r.rig, task.ID)
	if who["Did edit_file"] != "Developer" || who["Did run_task"] != "QA" {
		t.Fatalf("each step names the teammate who did it: %v", who)
	}

	proof := r.task(t, task.ID).Proof
	if proof == nil || len(proof.Checks) != 1 || proof.Checks[0].By != "QA" {
		t.Fatalf("the proof says who ran the check: %+v", proof)
	}

	if lead := r.lastEvent(task.ID, domain.EventReceipt); lead.Agent != "Orchestrator" {
		t.Fatalf("the lead's own steps carry its name too: %+v", lead)
	}
}

// stepOwners maps what each step says to who did it, and checks every handoff names the model.
func stepOwners(t *testing.T, r *rig, taskID string) map[string]string {
	t.Helper()

	events, _ := r.mem.EventsAfter(context.Background(), taskID, 0)
	who := map[string]string{}

	for _, event := range events {
		if event.Kind == domain.EventAction {
			who[event.Text] = event.Agent
		}

		if event.Kind == domain.EventHandoff && event.Action == "assigned" && event.Model == "" {
			t.Errorf("a handoff says which model the teammate works with: %+v", event)
		}
	}

	return who
}
