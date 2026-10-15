package application_test

import (
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestSwitchedOffToolIsNotOfferedAndNotRun(t *testing.T) {
	t.Parallel()

	r := newRig(nil, call("c1", "run_command", map[string]string{"command": "ls"}), finish("Done"))
	r.gate.off["run_command"] = true
	r.tools.risks["run_command"] = domain.RiskMedium
	task := r.start(t, "List the files")
	r.run(task.ID)

	for _, def := range r.model.requests[0].Tools {
		if def.Name == "run_command" {
			t.Fatal("a tool the person switched off must not be offered to the model")
		}
	}

	last := r.model.requests[1].Messages[len(r.model.requests[1].Messages)-1].Blocks[0]
	if len(r.tools.executed) != 0 || !last.IsError || r.task(t, task.ID).Status == domain.StatusNeedsInput {
		t.Fatalf("a switched-off tool must be refused without asking: %+v executed=%v", last, r.tools.executed)
	}
}

func TestSkippingApprovalsRunsEnabledToolsWithoutAsking(t *testing.T) {
	t.Parallel()

	r := newRig(nil, call("c1", "delete_file", map[string]string{"path": "old.txt"}), finish("Cleaned up"))
	r.gate.skip = true
	r.tools.risks["delete_file"] = domain.RiskMedium
	r.tools.results["delete_file"] = application.ToolResult{Output: "ok", Files: []string{"old.txt"}}
	task := r.start(t, "Remove the old file")
	r.run(task.ID)

	if !slices.Equal(r.tools.executed, []string{"delete_file"}) || r.task(t, task.ID).Status != domain.StatusReview {
		t.Fatalf("with approvals skipped the step runs at once: %v", r.tools.executed)
	}
}

func TestSkippingApprovalsDoesNotOverrideASwitch(t *testing.T) {
	t.Parallel()

	r := newRig(nil, call("c1", "delete_file", map[string]string{"path": "old.txt"}), finish("Left it"))
	r.gate.skip, r.gate.off["delete_file"] = true, true
	r.tools.risks["delete_file"] = domain.RiskMedium
	task := r.start(t, "Remove the old file")
	r.run(task.ID)

	if len(r.tools.executed) != 0 {
		t.Fatalf("a switched-off tool never runs, even when approvals are skipped: %v", r.tools.executed)
	}
}
