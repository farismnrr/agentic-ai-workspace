package application_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func finish(summary string, checks ...string) step {
	return call("fin", "finish", map[string]any{"summary": summary, "checks": checks})
}

func TestWorkReachesReviewWithPlanAndVerifiedProof(t *testing.T) {
	t.Parallel()

	plan := map[string]any{"steps": []map[string]string{
		{"title": "Look at the page", "state": "done"}, {"title": "Fix the button", "state": "active"},
	}}
	r := newRig(nil,
		call("c1", "update_plan", plan),
		call("c2", "edit_file", map[string]string{"path": "a.html"}),
		call("c3", "run_task", map[string]string{"name": "test"}),
		finish("Fixed the button", "Looked at the page"),
	)
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"a.html"}, Undo: []byte("u")}
	r.tools.results["run_task"] = application.ToolResult{Output: "pass", Run: &domain.RunInfo{Name: "test"}}

	task := r.start(t, "Fix the sign-in button")
	r.run(task.ID)

	done := r.task(t, task.ID)
	if done.Status != domain.StatusReview || len(done.Plan) != 2 || done.Summary != "Fixed the button" {
		t.Fatalf("expected review with plan and summary: %+v", done)
	}

	want := []domain.EventKind{domain.EventRequest, domain.EventAction, domain.EventAction, domain.EventReceipt}
	if got := r.kinds(task.ID); !slices.Equal(got, want) {
		t.Fatalf("thread = %v, want %v", got, want)
	}

	checks := done.Proof.Checks
	if !done.Proof.Proven || len(checks) != 2 || !checks[0].Verified || checks[1].Verified {
		t.Fatalf("a passing task is verified proof, the agent's own claim is not: %+v", done.Proof)
	}
}

func TestClaimsWithoutARunAreNeverProof(t *testing.T) {
	t.Parallel()

	r := newRig(nil, call("c1", "edit_file", map[string]string{"path": "a"}), finish("All good", "I tested everything"))
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"a"}, Undo: []byte("u")}
	task := r.start(t, "Check things")
	r.run(task.ID)

	proof := r.task(t, task.ID).Proof
	if proof.Proven || proof.Checks[0].Verified {
		t.Fatalf("a claim alone must not count as proof: %+v", proof)
	}
}

func TestAFailingTaskAfterTheLastChangeBlocksProof(t *testing.T) {
	t.Parallel()

	r := newRig(nil, call("c1", "run_task", map[string]string{"name": "test"}), finish("Done"))
	r.tools.results["run_task"] = application.ToolResult{Run: &domain.RunInfo{Name: "test", ExitCode: 1}}
	task := r.start(t, "Run tests")
	r.run(task.ID)

	if proof := r.task(t, task.ID).Proof; proof.Proven || proof.Rows[2].Value != "A check failed" {
		t.Fatalf("a failed check must not be proven: %+v", proof)
	}
}

func TestStepLimitStopsARunawayAgent(t *testing.T) {
	t.Parallel()

	steps := make([]step, 20)
	for i := range steps {
		steps[i] = call("c"+string(rune('a'+i)), "read_file", map[string]string{"path": "a"})
	}

	r := newRig(nil, steps...)
	task := r.start(t, "Loop forever")
	r.run(task.ID)

	if r.task(t, task.ID).Status != domain.StatusFailed || len(r.model.requests) != 10 {
		t.Fatalf("the run must stop at the step limit: %d calls", len(r.model.requests))
	}
}

func TestTokenBudgetStopsTheRun(t *testing.T) {
	t.Parallel()

	big := call("c1", "read_file", map[string]string{"path": "a"})
	big.reply.Usage = domain.Usage{Input: 900, Output: 200}
	r := newRig(nil, big, finish("never reached"))
	task := r.start(t, "Spend a lot")
	r.run(task.ID)

	if r.task(t, task.ID).Status != domain.StatusFailed || len(r.model.requests) != 1 {
		t.Fatalf("the budget must stop the run after the expensive call")
	}
}

func TestRetryableModelErrorsAreRetriedAndPermanentOnesFail(t *testing.T) {
	t.Parallel()

	retried := newRig(nil, failing("rate limited", true), failing("busy", true), finish("Done"))
	task := retried.start(t, "Try")
	retried.run(task.ID)

	if retried.task(t, task.ID).Status != domain.StatusDone {
		t.Fatal("transient errors should be retried")
	}

	broken := newRig(nil, failing("invalid api key", false))
	task = broken.start(t, "Try")
	broken.run(task.ID)

	if broken.task(t, task.ID).Status != domain.StatusFailed ||
		broken.lastEvent(task.ID, domain.EventError).Text != "The AI model said: invalid api key" {
		t.Fatalf("a permanent error must fail with the provider's message: %+v", broken.lastEvent(task.ID, domain.EventError))
	}
}

func TestCancellingStopsTheRunAndKeepsItResumable(t *testing.T) {
	t.Parallel()

	r := newRig(blockingModel{})
	task := r.start(t, "Slow")
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	r.runner.Run(ctx, task.ID)

	got, stoppedText := r.task(t, task.ID).Status, r.lastEvent(task.ID, domain.EventError).Text
	if got != domain.StatusFailed || stoppedText != "Stopped." {
		t.Fatalf("cancelled runs end as stopped: %v", got)
	}
}

func TestAQuestionIsAnsweredInTheThreadWithoutReview(t *testing.T) {
	t.Parallel()

	reply := step{reply: application.Reply{
		Blocks: []domain.Block{{Kind: domain.BlockText, Text: "It is a bakery website."}},
		Usage:  domain.Usage{Input: 10, Output: 5},
	}}
	r := newRig(nil, reply)
	task := r.start(t, "What is this project?")
	r.run(task.ID)

	done := r.task(t, task.ID)
	want := []domain.EventKind{domain.EventRequest, domain.EventMessage}

	if done.Status != domain.StatusDone || done.Summary != "It is a bakery website." || done.Proof != nil {
		t.Fatalf("an answer needs no review and no proof: %+v", done)
	}

	if got := r.kinds(task.ID); !slices.Equal(got, want) {
		t.Fatalf("thread = %v, want %v", got, want)
	}
}

func TestFinishingWithoutChangingAnythingIsAlsoJustAnAnswer(t *testing.T) {
	t.Parallel()

	r := newRig(nil, call("c1", "read_file", map[string]string{"path": "a"}), finish("It has two pages.", "Looked at a"))
	task := r.start(t, "How many pages?")
	r.run(task.ID)

	if got := r.task(t, task.ID).Status; got != domain.StatusDone {
		t.Fatalf("reading alone must not ask for a review, got %s", got)
	}

	if kinds := r.kinds(task.ID); slices.Contains(kinds, domain.EventReceipt) {
		t.Fatalf("an answer has no receipt: %v", kinds)
	}
}
