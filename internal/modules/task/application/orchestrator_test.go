package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestStartOrchestratorCreatesQueuedTeamTask(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()})
	task, err := r.service.StartOrchestrator(context.Background(), "u1", "proj", "Build the menu")
	if err != nil {
		t.Fatal(err)
	}
	if !task.Team || task.Source != "" || task.Status != domain.StatusQueued {
		t.Fatalf("expected a queued, owner-run team task, got %+v", task)
	}
	if len(r.disp.enqueued) != 1 || r.disp.enqueued[0] != task.ID {
		t.Fatalf("task was not queued asynchronously: %v", r.disp.enqueued)
	}
}

func TestOrchestratorOperationsEnforceTaskBoundary(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()})
	task := r.startPlanned(t, "Build the menu")
	ctx := context.Background()
	assertOtherOwnerCannotReadTask(t, r, ctx, task.ID)
	assertOtherProjectCannotReadTask(t, r, ctx, task.ID)
	assertSoloTaskIsNotOrchestrator(t, r, ctx, task.ID)
	assertConnectedTaskIsNotOrchestrator(t, r, ctx, task.ID)
}

func assertOtherOwnerCannotReadTask(t *testing.T, r *teamRig, ctx context.Context, taskID string) {
	t.Helper()
	if _, err := r.service.GetOrchestrator(ctx, "other-user", "proj", taskID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("another owner must not read the task: %v", err)
	}
}

func assertOtherProjectCannotReadTask(t *testing.T, r *teamRig, ctx context.Context, taskID string) {
	t.Helper()
	if _, err := r.service.GetOrchestrator(ctx, "u1", "other-project", taskID); !errors.Is(err, domain.ErrNoProject) {
		t.Errorf("a project not owned by the user must be rejected: %v", err)
	}
	if err := r.mem.update(taskID, func(current *domain.Task) { current.ProjectID = "different-project" }); err != nil {
		t.Fatal(err)
	}
	if _, err := r.service.GetOrchestrator(ctx, "u1", "proj", taskID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("a task from another project must not be exposed: %v", err)
	}
	if err := r.mem.update(taskID, func(current *domain.Task) { current.ProjectID = "proj" }); err != nil {
		t.Fatal(err)
	}
}

func assertSoloTaskIsNotOrchestrator(t *testing.T, r *teamRig, ctx context.Context, taskID string) {
	t.Helper()
	if err := r.mem.update(taskID, func(current *domain.Task) { current.Team = false }); err != nil {
		t.Fatal(err)
	}
	if _, err := r.service.GetOrchestrator(ctx, "u1", "proj", taskID); !errors.Is(err, domain.ErrWrongState) {
		t.Errorf("Solo tasks must not be exposed: %v", err)
	}
}

func assertConnectedTaskIsNotOrchestrator(t *testing.T, r *teamRig, ctx context.Context, taskID string) {
	t.Helper()
	if err := r.mem.update(taskID, func(current *domain.Task) {
		current.Team = true
		current.Source = "Connected app"
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.service.SendOrchestratorMessage(
		ctx, "u1", "proj", taskID, "Continue",
	); !errors.Is(err, domain.ErrWrongState) {
		t.Errorf("connected-app threads must not be exposed: %v", err)
	}
	if _, err := r.service.GetOrchestrator(ctx, "u1", "proj", taskID); !errors.Is(err, domain.ErrWrongState) {
		t.Errorf("connected-app threads must not be readable: %v", err)
	}
	if err := r.service.RetryOrchestrator(ctx, "u1", "proj", taskID); !errors.Is(err, domain.ErrWrongState) {
		t.Errorf("connected-app threads must not be retried: %v", err)
	}
}

func TestOrchestratorMessageUsesExistingFollowUpRules(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()})
	task := r.startPlanned(t, "Build the menu")
	ctx := context.Background()

	for _, test := range []struct {
		want   error
		status domain.Status
	}{
		{status: domain.StatusWorking, want: domain.ErrBusy},
		{status: domain.StatusNeedsInput, want: domain.ErrNeedsAnswer},
		{status: domain.StatusQueued, want: domain.ErrBusy},
	} {
		if err := r.mem.SetStatus(ctx, task.ID, test.status, time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := r.service.SendOrchestratorMessage(
			ctx, "u1", "proj", task.ID, "Continue",
		); !errors.Is(err, test.want) {
			t.Errorf("status %s: got %v, want %v", test.status, err, test.want)
		}
	}

	archivedAt := time.Now()
	if err := r.mem.update(task.ID, func(current *domain.Task) {
		current.Status = domain.StatusDone
		current.ArchivedAt = &archivedAt
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.service.SendOrchestratorMessage(
		ctx, "u1", "proj", task.ID, "Continue",
	); !errors.Is(err, domain.ErrWrongState) {
		t.Errorf("archived tasks must not accept messages: %v", err)
	}

	if err := r.mem.update(task.ID, func(current *domain.Task) {
		current.Status = domain.StatusReview
		current.ArchivedAt = nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.service.SendOrchestratorMessage(
		ctx, "u1", "proj", task.ID, "Continue",
	); !errors.Is(err, domain.ErrWrongState) {
		t.Fatalf("Orchestrator messages must not accept work awaiting review: %v", err)
	}
	if got := r.task(t, task.ID).Status; got != domain.StatusReview {
		t.Fatalf("task awaiting the person's review changed status: %s", got)
	}
}

func TestOrchestratorMessageCannotResolveApproval(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()})
	task := r.startPlanned(t, "Build the menu")
	approval, appendErr := r.mem.AppendEvent(context.Background(), task.ID, domain.Event{
		Kind: domain.EventApproval, Text: "May I change files?",
	})
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if err := r.mem.SetStatus(context.Background(), task.ID, domain.StatusNeedsInput, time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := r.service.SendOrchestratorMessage(
		context.Background(), "u1", "proj", task.ID, "Yes, continue",
	); !errors.Is(err, domain.ErrNeedsAnswer) {
		t.Fatalf("an MCP message must not act as approval: %v", err)
	}
	got, findErr := r.mem.FindEvent(context.Background(), task.ID, approval.ID)
	if findErr != nil || got.Resolved != "" {
		t.Fatalf("the pending approval was changed: event=%+v err=%v", got, findErr)
	}
}

func TestRetryOrchestratorOnlyQueuesFailedTask(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{developer()})
	task := r.startPlanned(t, "Build the menu")
	ctx := context.Background()

	if err := r.service.RetryOrchestrator(ctx, "u1", "proj", task.ID); !errors.Is(err, domain.ErrWrongState) {
		t.Fatalf("non-failed task was retried: %v", err)
	}
	if err := r.mem.SetStatus(ctx, task.ID, domain.StatusFailed, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := r.service.RetryOrchestrator(ctx, "u1", "proj", task.ID); err != nil {
		t.Fatalf("failed task was not retried: %v", err)
	}
	got, err := r.service.GetOrchestrator(ctx, "u1", "proj", task.ID)
	if err != nil || got.Status != domain.StatusQueued || got.ID != task.ID {
		t.Fatalf("retry should requeue the same task: %+v, %v", got, err)
	}
}
