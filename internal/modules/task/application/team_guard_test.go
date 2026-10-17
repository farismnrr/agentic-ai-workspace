package application_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestTeammateApprovalPausesTheTaskAndResumesTheSameTeammate(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()},
		delegate("d1", "developer", "Change it"),
		call("c1", "edit_file", map[string]string{"path": "a.txt"}),
		report("r1", "Changed"),
		finish("Done"),
	)
	r.tools.risks["edit_file"] = domain.RiskLow
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"a.txt"}}
	task := r.startTeam(t, "Change the page")
	r.run(task.ID)

	approval := r.lastEvent(task.ID, domain.EventApproval)
	if r.task(t, task.ID).Status != domain.StatusNeedsInput || approval.Agent != "Developer" ||
		r.delegations.status(task.ID) != application.DelegationWaiting || len(r.tools.executed) != 0 {
		t.Fatalf("the teammate asks, named, and the task waits: %+v", approval)
	}

	yes := true
	answer := application.Answer{Allow: &yes}
	if err := r.service.Answer(context.Background(), "u1", task.ID, approval.ID, answer); err != nil {
		t.Fatal(err)
	}

	r.run(task.ID)

	got := r.task(t, task.ID).Status
	if got != domain.StatusReview || !slices.Equal(r.tools.executed, []string{"edit_file"}) {
		t.Fatalf("after the yes the same teammate carries on and the step runs once: %v %v", got, r.tools.executed)
	}

	if len(r.model.requests) != 4 || r.model.requests[2].Model != "dev-model" {
		t.Fatalf("the resumed teammate keeps its model: %d requests", len(r.model.requests))
	}
}

func TestRoleCannotUseToolsOutsideItsAccess(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{researcher()},
		delegate("d1", "researcher", "Look around"),
		call("c1", "run_command", map[string]string{"command": "rm -rf /"}),
		call("c2", "ask_user", map[string]string{"question": "Hello?"}),
		report("r1", "Could not"),
		finish("Done"),
	)
	r.tools.risks["run_command"] = domain.RiskMedium
	task := r.startTeam(t, "Look")
	r.run(task.ID)

	if len(r.tools.executed) != 0 || r.lastEvent(task.ID, domain.EventApproval).ID != "" ||
		r.lastEvent(task.ID, domain.EventQuestion).ID != "" {
		t.Fatalf("a read-only teammate can neither run commands nor ask the person: %v", r.tools.executed)
	}

	if !resultFor(r.model.requests[2], "c1").IsError || !resultFor(r.model.requests[3], "c2").IsError {
		t.Fatal("both refusals must reach the teammate as errors")
	}
}

func TestUnknownRoleIsAnErrorTheOrchestratorCanReadAndAnswerAround(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()},
		delegate("d1", "plumber", "Fix the pipes"), finish("Handled it myself"))
	task := r.startTeam(t, "Fix it")
	r.run(task.ID)

	got := resultFor(r.model.requests[1], "d1")
	if !got.IsError || !strings.Contains(got.Text, "developer") || r.task(t, task.ID).Status == domain.StatusFailed {
		t.Fatalf("the orchestrator is told who is available and the task goes on: %+v", got)
	}
}

func TestTeammateStopsAfterItsOwnShareOfSteps(t *testing.T) {
	t.Parallel()

	r := newTeamRig(func(l *application.Limits) { l.DelegateSteps = 2 }, []application.AgentRole{developer()},
		delegate("d1", "developer", "Loop"),
		call("c1", "read_file", map[string]string{"path": "a"}),
		call("c2", "read_file", map[string]string{"path": "b"}),
		finish("Gave up on that part"),
	)
	task := r.startTeam(t, "Do it")
	r.run(task.ID)

	got := resultFor(r.model.requests[3], "d1")
	ended := r.delegations.status(task.ID) == application.DelegationFailed
	ended = ended && r.task(t, task.ID).Status == domain.StatusDone

	if !got.IsError || !strings.Contains(got.Text, "stopped after 2 steps") || !ended {
		t.Fatalf("a teammate that loops fails alone and the orchestrator carries on: %+v", got)
	}
}

func TestTeamShareOneStepBudget(t *testing.T) {
	t.Parallel()

	r := newTeamRig(func(l *application.Limits) { l.TeamSteps = 3 }, []application.AgentRole{developer()},
		delegate("d1", "developer", "Loop"),
		call("c1", "read_file", map[string]string{"path": "a"}),
		call("c2", "read_file", map[string]string{"path": "b"}),
		finish("Never reached"),
	)
	task := r.startTeam(t, "Do it")
	r.run(task.ID)

	if got := r.task(t, task.ID).Status; got != domain.StatusFailed ||
		!strings.Contains(r.lastEvent(task.ID, domain.EventError).Text, "stopped after 3 steps") {
		t.Fatalf("orchestrator and teammates draw from one budget: %v", got)
	}
}

func TestTeammateChecksCountAsProofAndUsageGoesToTheirModel(t *testing.T) {
	t.Parallel()

	qa := application.AgentRole{Key: "qa", Name: "QA", ProviderID: "prov", Model: "qa-model", Tools: []string{"run_task"}}
	r := newTeamRig(nil, []application.AgentRole{developer(), qa},
		delegate("d1", "developer", "Change it"),
		call("c1", "edit_file", map[string]string{"path": "a.txt"}),
		report("r1", "Changed"),
		delegate("d2", "qa", "Run the tests"),
		call("c2", "run_task", map[string]string{"name": "test"}),
		report("r2", "Passed"),
		finish("Done"),
	)
	r.gate.skip = true
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"a.txt"}}
	r.tools.results["run_task"] = application.ToolResult{Output: "ok", Run: &domain.RunInfo{Name: "test"}}
	task := r.startTeam(t, "Change and test")
	r.run(task.ID)

	got := r.task(t, task.ID)
	if got.Proof == nil || !got.Proof.Proven {
		t.Fatalf("a check the QA teammate ran after the last change proves the work: %+v", got.Proof)
	}

	models := map[string]bool{}
	for _, usage := range r.mem.usage {
		models[usage.Model] = true
	}

	if !models["dev-model"] || !models["qa-model"] || !models["model-x"] {
		t.Fatalf("each agent's tokens are billed to its own model: %v", models)
	}
}
