package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestCreateValidatesAndQueuesTheAgent(t *testing.T) {
	t.Parallel()

	r := newRig(nil)
	ctx := context.Background()

	create := func(project, prompt string) error {
		_, err := r.service.Create(ctx, "u1", application.CreateInput{ProjectID: project, Prompt: prompt})
		if err != nil {
			return fmt.Errorf("create: %w", err)
		}

		return nil
	}

	if err := create("proj", "  "); !errors.Is(err, domain.ErrBadRequest) {
		t.Errorf("a blank request must be refused, got %v", err)
	}

	if err := create("nope", "Hi"); !errors.Is(err, domain.ErrNoProject) {
		t.Errorf("an unknown project must be refused, got %v", err)
	}

	r.models.none = true

	if err := create("proj", "Hi"); !errors.Is(err, domain.ErrNoModel) {
		t.Errorf("without a model the person is told to add one, got %v", err)
	}

	r.models.none = false
	task := r.start(t, "Make my menu page\nwith photos")

	if task.Title != "Make my menu page" || len(r.disp.enqueued) != 1 || r.kinds(task.ID)[0] != domain.EventRequest {
		t.Fatalf("the request is recorded and the agent queued: %+v %v", task, r.disp.enqueued)
	}
}

func TestFollowUpRulesByStatus(t *testing.T) {
	t.Parallel()

	r := newRig(nil)
	task := r.start(t, "Hello")
	ctx := context.Background()

	for status, want := range map[domain.Status]error{
		domain.StatusWorking: domain.ErrBusy, domain.StatusNeedsInput: domain.ErrNeedsAnswer, domain.StatusReview: nil,
	} {
		_ = r.mem.SetStatus(ctx, task.ID, status, r.mem.tasks[task.ID].UpdatedAt)

		if err := r.service.FollowUp(ctx, "u1", task.ID, "Also the footer"); !errors.Is(err, want) {
			t.Errorf("status %s: got %v, want %v", status, err, want)
		}
	}
}

func TestAFollowUpKeepsToolResultsAheadOfTheNewText(t *testing.T) {
	t.Parallel()

	r := newRig(nil, finish("First"), finish("Second"))
	task := r.start(t, "Do it")
	r.run(task.ID)

	if err := r.service.FollowUp(context.Background(), "u1", task.ID, "And the footer please"); err != nil {
		t.Fatal(err)
	}

	r.run(task.ID)

	last := r.model.requests[1].Messages[len(r.model.requests[1].Messages)-1]
	if last.Blocks[0].Kind != domain.BlockToolResult || last.Blocks[1].Text != "And the footer please" {
		t.Fatalf("tool results must come first in the user turn: %+v", last.Blocks)
	}

	if r.task(t, task.ID).Status != domain.StatusDone {
		t.Fatal("a follow-up that changes nothing is answered, not sent to review")
	}
}

func TestStateChangesAreRestrictedAndOwned(t *testing.T) {
	t.Parallel()

	r := newRig(nil)
	task := r.start(t, "Hello")
	ctx := context.Background()

	if err := r.service.Accept(ctx, "u1", task.ID); !errors.Is(err, domain.ErrWrongState) {
		t.Errorf("only reviewed work can be accepted, got %v", err)
	}

	_ = r.mem.SetStatus(ctx, task.ID, domain.StatusReview, task.UpdatedAt)

	if err := r.service.Accept(ctx, "u1", task.ID); err != nil || r.task(t, task.ID).Status != domain.StatusDone {
		t.Errorf("accepting reviewed work: %v", err)
	}

	if err := r.service.Accept(ctx, "intruder", task.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("another person must not reach the task, got %v", err)
	}

	if err := r.service.Retry(ctx, "u1", task.ID); !errors.Is(err, domain.ErrWrongState) {
		t.Errorf("only stopped tasks can continue, got %v", err)
	}
}

func TestUndoReversesAStepOnceAndNeverWhileRunning(t *testing.T) {
	t.Parallel()

	r := newRig(nil, call("c1", "edit_file", map[string]string{"path": "a"}), finish("Done"))
	r.tools.results["edit_file"] = application.ToolResult{Output: "ok", Files: []string{"a"}, Undo: []byte("blob")}
	task := r.start(t, "Edit")
	r.run(task.ID)

	ctx := context.Background()
	step := r.lastEvent(task.ID, domain.EventAction)

	if err := r.service.UndoChange(ctx, "u1", task.ID, step.ID); err != nil || string(r.tools.undone[0]) != "blob" {
		t.Fatalf("undo should hand the blob to the tools: %v %v", err, r.tools.undone)
	}

	if err := r.service.UndoChange(ctx, "u1", task.ID, step.ID); !errors.Is(err, domain.ErrWrongState) {
		t.Errorf("a step can be undone once, got %v", err)
	}
}

func TestRecoverMarksInterruptedTasksSoNothingIsStuck(t *testing.T) {
	t.Parallel()

	r := newRig(nil)
	task := r.start(t, "Hello")
	_ = r.mem.SetStatus(context.Background(), task.ID, domain.StatusWorking, task.UpdatedAt)

	if err := r.service.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}

	if r.task(t, task.ID).Status != domain.StatusFailed || r.lastEvent(task.ID, domain.EventError).Hint == "" {
		t.Fatal("interrupted tasks fail with a way to continue")
	}
}
