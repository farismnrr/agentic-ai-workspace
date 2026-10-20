package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func failingRuns(n int) []step {
	steps := []step{delegate("d1", "developer", "Make the tests pass")}
	for i := range n {
		steps = append(steps, call("c"+string(rune('1'+i)), "run_task", map[string]string{"name": "test"}))
	}

	return steps
}

func stuckRig(extra ...step) *teamRig {
	r := newTeamRig(nil, []application.AgentRole{developer()}, append(failingRuns(6), extra...)...)
	r.tools.results["run_task"] = application.ToolResult{Output: "tests failed: cannot find module x", IsError: true}

	return r
}

func TestATeammateWhoKeepsFailingIsToldToReportBeingStuck(t *testing.T) {
	t.Parallel()

	r := stuckRig(finish("Done"))
	r.run(r.startTeam(t, "Fix the tests").ID)

	nudged := 0

	for _, req := range r.model.requests {
		for _, m := range req.Messages {
			for _, b := range m.Blocks {
				if strings.Contains(b.Text, "Stuck:") && strings.Contains(b.Text, "call report now") {
					nudged++
				}
			}
		}
	}

	if nudged == 0 {
		t.Fatal("after three failures in a row the teammate must be told to report that it is stuck")
	}
}

func TestATeammateWhoNeverGetsPastItIsStoppedAndTheLeadIsToldWhatToDo(t *testing.T) {
	t.Parallel()

	r := stuckRig(finish("Done"))
	task := r.startTeam(t, "Fix the tests")
	r.run(task.ID)

	if r.delegations.status(task.ID) != application.DelegationFailed {
		t.Fatalf("six failures in a row must end the job, got %q", r.delegations.status(task.ID))
	}

	d, _, _ := r.delegations.FindDelegation(t.Context(), task.ID, "d1")
	if !strings.Contains(d.Report, "cannot find module x") || !strings.Contains(d.Report, "Researcher") {
		t.Fatalf("the lead must get the last error and be told to ask the Researcher: %q", d.Report)
	}

	if len(r.tools.executed) != 6 {
		t.Fatalf("the teammate must not go on after giving up: %d steps", len(r.tools.executed))
	}
}

func TestAStepThatWorksStartsTheCountAgain(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()},
		delegate("d1", "developer", "Make the tests pass"),
		call("c1", "run_task", map[string]string{}), call("c2", "run_task", map[string]string{}),
		call("c3", "read_file", map[string]string{"path": "a"}),
		call("c4", "run_task", map[string]string{}), call("c5", "run_task", map[string]string{}),
		report("r1", "Done"), finish("Done"),
	)
	r.tools.results["run_task"] = application.ToolResult{Output: "failed", IsError: true}
	task := r.startTeam(t, "Fix the tests")
	r.run(task.ID)

	if r.delegations.status(task.ID) != application.DelegationDone {
		t.Fatalf("failures with a success between them are not a streak: %q", r.delegations.status(task.ID))
	}
}
