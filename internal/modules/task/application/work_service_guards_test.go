package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestRetryAndAnswerRefuseConnectedAppTasks(t *testing.T) {
	r := newRig(nil)
	ctx := context.Background()
	id, err := r.service.Record(ctx, "u1", application.RecordInput{
		ProjectID: "proj", Source: "ChatGPT", Event: domain.Event{Kind: domain.EventRequest, Text: "Do this"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = r.mem.SetStatus(ctx, id, domain.StatusFailed, time.Now())
	if err := r.service.Retry(ctx, "u1", id); !errors.Is(err, domain.ErrWrongState) || len(r.disp.enqueued) != 0 {
		t.Fatalf("external task was requeued: err=%v queue=%v", err, r.disp.enqueued)
	}
	if err := r.service.Answer(ctx, "u1", id, "missing", application.Answer{}); !errors.Is(err, domain.ErrWrongState) {
		t.Fatalf("external task accepted an internal answer: %v", err)
	}
}

func TestRetryAnswerAndContinueStillQueueInternalTasks(t *testing.T) {
	r := newRig(nil)
	ctx := context.Background()
	failed := r.start(t, "Retry this")
	_ = r.mem.SetStatus(ctx, failed.ID, domain.StatusFailed, time.Now())
	if err := r.service.Retry(ctx, "u1", failed.ID); err != nil {
		t.Fatalf("retry internal task: %v", err)
	}
	questionTask := r.start(t, "Answer this")
	question, err := r.mem.AppendEvent(ctx, questionTask.ID, domain.Event{Kind: domain.EventQuestion, Text: "Choose"})
	if err != nil {
		t.Fatal(err)
	}
	_ = r.mem.SetStatus(ctx, questionTask.ID, domain.StatusNeedsInput, time.Now())
	if err := r.service.Answer(ctx, "u1", questionTask.ID, question.ID, application.Answer{Choice: "Yes"}); err != nil {
		t.Fatalf("answer internal task: %v", err)
	}
	review := r.start(t, "Continue this")
	_ = r.mem.SetStatus(ctx, review.ID, domain.StatusReview, time.Now())
	if err := r.service.FollowUp(ctx, "u1", review.ID, "One more thing"); err != nil {
		t.Fatalf("continue internal task: %v", err)
	}
	if len(r.disp.enqueued) != 6 {
		t.Fatalf("internal operations were not all enqueued; queue=%v", r.disp.enqueued)
	}
}

func TestRecoverySkipsWorkAndLegacyCallsDoNotJoinIt(t *testing.T) {
	r := newRig(nil)
	ctx := context.Background()
	workTask := domain.Task{
		ID: "work-task", UserID: "u1", ProjectID: "proj", Source: "ChatGPT",
		WorkID: "work-id", Status: domain.StatusWorking, UpdatedAt: time.Now(), Plan: []domain.PlanStep{},
	}
	if err := r.mem.Create(ctx, workTask); err != nil {
		t.Fatal(err)
	}
	if err := r.service.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	if got := r.task(t, workTask.ID).Status; got != domain.StatusWorking || len(r.disp.enqueued) != 0 {
		t.Fatalf("recovery changed active work state to %s or enqueued it: %v", got, r.disp.enqueued)
	}
	first, err := r.service.Record(ctx, "u1", application.RecordInput{
		ProjectID: "proj", Source: "ChatGPT", Event: domain.Event{Kind: domain.EventMessage, Text: "Legacy call"},
	})
	if err != nil || first == workTask.ID {
		t.Fatalf("legacy call joined an explicit work: id=%s err=%v", first, err)
	}
	second, err := r.service.Record(ctx, "u1", application.RecordInput{
		ProjectID: "proj", Source: "ChatGPT", Event: domain.Event{Kind: domain.EventMessage, Text: "Next legacy call"},
	})
	if err != nil || second != first || len(r.disp.enqueued) != 0 {
		t.Fatalf("legacy grouping or enqueue behavior changed: first=%s second=%s err=%v queue=%v",
			first, second, err, r.disp.enqueued)
	}
}

type stopRecorder struct{ user, client, project, work string }

func (s *stopRecorder) Stop(_ context.Context, user, client, project, work string) error {
	s.user, s.client, s.project, s.work = user, client, project, work
	return nil
}

func TestCancelDelegatesWorkTasksToWorkStop(t *testing.T) {
	r := newRig(nil)
	stop := &stopRecorder{}
	r.service.UseWorkServices(stop, nil)
	task := domain.Task{
		ID: "work-task", UserID: "u1", ProjectID: "proj", Source: "ChatGPT",
		WorkID: "work-id", Status: domain.StatusWorking, Plan: []domain.PlanStep{},
	}
	if err := r.mem.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if err := r.service.Cancel(context.Background(), "u1", task.ID); err != nil {
		t.Fatal(err)
	}
	if stop.user != "u1" || stop.client != "ChatGPT" || stop.project != "proj" || stop.work != "work-id" {
		t.Fatalf("cancel did not target linked work: %+v", stop)
	}
}

func TestUndoAllowsSafeExternalChangeWhileWorkIsOpen(t *testing.T) {
	r := newRig(nil)
	ctx := context.Background()
	task := domain.Task{
		ID: "work-task", UserID: "u1", ProjectID: "proj", Source: "ChatGPT",
		WorkID: "work-id", Status: domain.StatusWorking, Plan: []domain.PlanStep{},
	}
	if err := r.mem.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	event, err := r.mem.AppendEvent(ctx, task.ID, domain.Event{
		Kind: domain.EventAction, Undo: []byte("safe change"), Work: &domain.WorkRef{ID: task.WorkID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.service.UndoChange(ctx, "u1", task.ID, event.ID); err != nil || len(r.tools.undone) != 1 {
		t.Fatalf("open external work change was not offered to guarded undo: err=%v undo=%q", err, r.tools.undone)
	}
}
