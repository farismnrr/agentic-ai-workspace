package application_test

import (
	"context"
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestRiskyStepWaitsForApprovalThenRunsOnce(t *testing.T) {
	t.Parallel()

	r := newRig(nil, call("c1", "delete_file", map[string]string{"path": "old.txt"}), finish("Cleaned up"))
	r.tools.risks["delete_file"] = domain.RiskMedium
	task := r.start(t, "Remove the old file", domain.ModeRisky)
	r.run(task.ID)

	approval := r.lastEvent(task.ID, domain.EventApproval)
	if r.task(t, task.ID).Status != domain.StatusNeedsInput || approval.Resolved != "" || len(r.tools.executed) != 0 {
		t.Fatalf("the agent must stop and ask before a risky step: %+v executed=%v", approval, r.tools.executed)
	}

	yes := true
	answer := application.Answer{Allow: &yes}
	if err := r.service.Answer(context.Background(), "u1", task.ID, approval.ID, answer); err != nil {
		t.Fatal(err)
	}

	r.run(task.ID)

	got := r.task(t, task.ID).Status
	if got != domain.StatusReview || !slices.Equal(r.tools.executed, []string{"delete_file"}) {
		t.Fatalf("after approval the step runs exactly once: %v %v", got, r.tools.executed)
	}
}

func TestDeniedStepIsNotRunAndTheModelIsTold(t *testing.T) {
	t.Parallel()

	r := newRig(nil, call("c1", "delete_file", map[string]string{"path": "old.txt"}), finish("Left it alone"))
	r.tools.risks["delete_file"] = domain.RiskMedium
	task := r.start(t, "Remove the old file", domain.ModeRisky)
	r.run(task.ID)

	no := false
	approval := r.lastEvent(task.ID, domain.EventApproval)
	_ = r.service.Answer(context.Background(), "u1", task.ID, approval.ID, application.Answer{Allow: &no})
	r.run(task.ID)

	last := r.model.requests[1].Messages[len(r.model.requests[1].Messages)-1].Blocks[0]
	if len(r.tools.executed) != 0 || !last.IsError || last.ID != "c1" {
		t.Fatalf("a denied step must be reported to the model as an error: %+v executed=%v", last, r.tools.executed)
	}
}

func TestOnlyAlwaysModeAsksBeforeEdits(t *testing.T) {
	t.Parallel()

	for mode, wantAsk := range map[domain.Mode]bool{domain.ModeRisky: false, domain.ModeAlways: true} {
		r := newRig(nil, call("c1", "edit_file", map[string]string{"path": "a"}), finish("Done"))
		r.tools.risks["edit_file"] = domain.RiskLow
		task := r.start(t, "Edit", mode)
		r.run(task.ID)

		asked := r.task(t, task.ID).Status == domain.StatusNeedsInput
		if asked != wantAsk {
			t.Errorf("mode %s: asked=%v, want %v", mode, asked, wantAsk)
		}
	}
}

func TestAskUserPausesAndTheAnswerReachesTheModel(t *testing.T) {
	t.Parallel()

	r := newRig(nil,
		call("c1", "ask_user", map[string]any{"question": "Which page?", "options": []string{"Home", "Menu"}}),
		finish("Done"),
	)
	task := r.start(t, "Update the page", domain.ModeRisky)
	r.run(task.ID)

	question := r.lastEvent(task.ID, domain.EventQuestion)
	if r.task(t, task.ID).Status != domain.StatusNeedsInput || len(question.Options) != 2 {
		t.Fatalf("the agent must wait for the answer: %+v", question)
	}

	_ = r.service.Answer(context.Background(), "u1", task.ID, question.ID, application.Answer{Choice: "Menu"})
	r.run(task.ID)

	if got := r.model.requests[1].Messages[2].Blocks[0].Text; got != "The person answered: Menu" {
		t.Fatalf("answer not passed on: %q", got)
	}
}
